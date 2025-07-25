package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto/ed25519"
	cmtflags "github.com/cometbft/cometbft/libs/cli/flags"
	nm "github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"
	cometbfttypes "github.com/cometbft/cometbft/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// state that the abci specifically relies on
type ABCIState struct {
	onGoingBlock     pgx.Tx
	finalizedTxs     []string
	lastRetainHeight int64
}

func NewABCIState(initialRetainHeight int64) *ABCIState {
	return &ABCIState{
		onGoingBlock:     nil,
		finalizedTxs:     []string{},
		lastRetainHeight: initialRetainHeight,
	}
}

var _ abcitypes.Application = (*Server)(nil)

// initializes the cometbft node and the abci application which is the server itself
// connects the local rpc instance to the abci application once successfully created
func (s *Server) startABCI(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.awaitEthReady:
	}
	s.logger.Info("starting abci")

	cometConfig := s.cometbftConfig
	pv := privval.LoadFilePV(
		cometConfig.PrivValidatorKeyFile(),
		cometConfig.PrivValidatorStateFile(),
	)

	nodeKey, err := p2p.LoadNodeKey(cometConfig.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load node's key: %v", err)
	}

	nodeLogger, err := cmtflags.ParseLogLevel(s.config.CometLogLevel, s.logger, "error")
	if err != nil {
		return fmt.Errorf("failed to parse log level: %v", err)
	}

	node, err := nm.NewNode(
		ctx,
		cometConfig,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(s),
		nm.DefaultGenesisDocProviderFunc(cometConfig),
		cfg.DefaultDBProvider,
		nm.DefaultMetricsProvider(cometConfig.Instrumentation),
		nodeLogger,
	)

	if err != nil {
		s.logger.Errorf("error creating node: %v", err)
		return fmt.Errorf("creating node: %v", err)
	}

	s.node = node
	s.rpc = local.New(s.node)
	close(s.awaitRpcReady)

	s.logger.Info("core CometBFT node starting")

	if err := s.node.Start(); err != nil {
		s.logger.Errorf("cometbft failed to start: %v", err)
		return err
	}

	<-ctx.Done()
	err = s.node.Stop()
	if err != nil {
		return err
	}
	return ctx.Err()
}

func (s *Server) Info(ctx context.Context, info *abcitypes.InfoRequest) (*abcitypes.InfoResponse, error) {
	latest, err := s.db.GetLatestAppState(ctx)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		// Log the error and return a default response
		s.logger.Errorf("Error retrieving app state: %v", err)
		return &abcitypes.InfoResponse{}, nil
	}

	s.logger.Infof("app starting at block %d with hash %s", latest.BlockHeight, hex.EncodeToString(latest.AppHash))

	res := &abcitypes.InfoResponse{
		LastBlockHeight:  latest.BlockHeight,
		LastBlockAppHash: latest.AppHash,
	}

	return res, nil
}

func (s *Server) Query(ctx context.Context, req *abcitypes.QueryRequest) (*abcitypes.QueryResponse, error) {
	return &abcitypes.QueryResponse{}, nil
}

func (s *Server) CheckTx(_ context.Context, check *abcitypes.CheckTxRequest) (*abcitypes.CheckTxResponse, error) {
	// check if protobuf event
	_, err := s.isValidSignedTransaction(check.Tx)
	if err == nil {
		return &abcitypes.CheckTxResponse{Code: abcitypes.CodeTypeOK}, nil
	}

	return &abcitypes.CheckTxResponse{Code: 1}, nil
}

func (s *Server) InitChain(_ context.Context, chain *abcitypes.InitChainRequest) (*abcitypes.InitChainResponse, error) {
	return &abcitypes.InitChainResponse{}, nil
}

func (s *Server) PrepareProposal(ctx context.Context, proposal *abcitypes.PrepareProposalRequest) (*abcitypes.PrepareProposalResponse, error) {
	proposalTxs := [][]byte{}

	shouldProposeNewRollup := s.shouldProposeNewRollup(ctx, proposal.Height)
	if shouldProposeNewRollup {
		rollupTx, err := s.createRollupTx(ctx, proposal.Time, proposal.Height)
		if err != nil {
			s.logger.Error("Failed to create rollup transaction", "error", err)
		} else {
			proposalTxs = append(proposalTxs, rollupTx)
		}
	}
	for _, mb := range proposal.Misbehavior {
		deregTx, err := s.createDeregisterTransaction(mb.Validator.Address)
		if err != nil {
			s.logger.Error("Failed to create deregistration transaction", "error", err)
		} else {
			proposalTxs = append(proposalTxs, deregTx)
		}
	}

	// keep batch at 1000 even if sla rollup occurs
	batch := 1000
	if shouldProposeNewRollup {
		batch = batch - 1
	}

	txMemBatch := s.mempl.GetBatch(batch, proposal.Height)

	// TODO: parallelize
	for _, tx := range txMemBatch {
		txBytes, err := proto.Marshal(tx)
		if err != nil {
			s.logger.Errorf("tx made it into prepare but couldn't be marshalled: %v", err)
			continue
		}
		valid, err := s.validateBlockTx(ctx, proposal.Time, proposal.Height, proposal.Misbehavior, txBytes)
		if err != nil {
			s.logger.Errorf("tx made it into prepare but couldn't be validated: %v", err)
			continue
		} else if !valid {
			s.logger.Errorf("invalid tx made it into prepare: %v", tx)
			continue
		}
		proposalTxs = append(proposalTxs, txBytes)
	}
	return &abcitypes.PrepareProposalResponse{Txs: proposalTxs}, nil
}

func (s *Server) ProcessProposal(ctx context.Context, proposal *abcitypes.ProcessProposalRequest) (*abcitypes.ProcessProposalResponse, error) {
	valid, err := s.validateBlockTxs(ctx, proposal.Time, proposal.Height, proposal.Misbehavior, proposal.Txs)
	if err != nil {
		s.logger.Error("Reporting unknown proposal status due to validation error", "error", err)
		return &abcitypes.ProcessProposalResponse{Status: abcitypes.PROCESS_PROPOSAL_STATUS_UNKNOWN}, err
	} else if !valid {
		return &abcitypes.ProcessProposalResponse{Status: abcitypes.PROCESS_PROPOSAL_STATUS_REJECT}, nil
	}
	return &abcitypes.ProcessProposalResponse{Status: abcitypes.PROCESS_PROPOSAL_STATUS_ACCEPT}, nil
}

func (s *Server) FinalizeBlock(ctx context.Context, req *abcitypes.FinalizeBlockRequest) (*abcitypes.FinalizeBlockResponse, error) {
	logger := s.logger
	state := s.abciState
	var txs = make([]*abcitypes.ExecTxResult, len(req.Txs))
	var validatorUpdatesMap = map[string]abcitypes.ValidatorUpdate{}

	// open in progres pg transaction
	s.startInProgressTx(ctx)

	if err := s.getDb().StoreBlock(ctx, db.StoreBlockParams{
		Height:    req.Height,
		Hash:      hex.EncodeToString(req.Hash),
		Proposer:  hex.EncodeToString(req.ProposerAddress),
		ChainID:   s.config.GenesisFile.ChainID,
		CreatedAt: s.db.ToPgxTimestamp(req.Time),
	}); err != nil {
		s.logger.Errorf("could not store block: %v", err)
	}

	for i, tx := range req.Txs {
		signedTx, err := s.isValidSignedTransaction(tx)
		if err == nil {
			// set tx to ok and set to not okay later if error occurs
			txs[i] = &abcitypes.ExecTxResult{Code: abcitypes.CodeTypeOK}

			txhash := s.toTxHash(signedTx)
			finalizedTx, err := s.finalizeTransaction(ctx, req, signedTx, txhash, req.Height)
			if err != nil {
				s.logger.Errorf("error finalizing event: %v", err)
				txs[i] = &abcitypes.ExecTxResult{Code: 2}
			} else if vr := signedTx.GetValidatorRegistration(); vr != nil { // TODO: delete legacy registration after chain rollover
				vrPubKey := ed25519.PubKey(vr.GetPubKey())
				vrAddr := vrPubKey.Address().String()
				if _, ok := validatorUpdatesMap[vrAddr]; !ok {
					validatorUpdatesMap[vrAddr] = abcitypes.ValidatorUpdate{
						Power:       vr.Power,
						PubKeyBytes: vr.PubKey,
						PubKeyType:  "ed25519",
					}
				}
			} else if att := signedTx.GetAttestation(); att != nil && att.GetValidatorRegistration() != nil {
				vr := att.GetValidatorRegistration()
				vrPubKey := ed25519.PubKey(vr.GetPubKey())
				vrAddr := vrPubKey.Address().String()
				if _, ok := validatorUpdatesMap[vrAddr]; !ok {
					validatorUpdatesMap[vrAddr] = abcitypes.ValidatorUpdate{
						Power:       vr.Power,
						PubKeyBytes: vr.PubKey,
						PubKeyType:  "ed25519",
					}
				}
			} else if att := signedTx.GetAttestation(); att != nil && att.GetValidatorDeregistration() != nil {
				vr := att.GetValidatorDeregistration()
				vrPubKey := ed25519.PubKey(vr.GetPubKey())
				vrAddr := vrPubKey.Address().String()
				// intentionally override any existing updates
				validatorUpdatesMap[vrAddr] = abcitypes.ValidatorUpdate{
					Power:       int64(0),
					PubKeyBytes: vr.PubKey,
					PubKeyType:  "ed25519",
				}
			} else if vd := signedTx.GetValidatorDeregistration(); vd != nil {
				vdPubKey := ed25519.PubKey(vd.GetPubKey())
				vdAddr := vdPubKey.Address().String()
				// intentionally override any existing updates
				validatorUpdatesMap[vdAddr] = abcitypes.ValidatorUpdate{
					Power:       int64(0),
					PubKeyBytes: vd.PubKey,
					PubKeyType:  "ed25519",
				}
			}

			if err := s.getDb().StoreTransaction(ctx, db.StoreTransactionParams{
				BlockID:     req.Height,
				Index:       int32(i),
				TxHash:      txhash,
				Transaction: tx,
				CreatedAt:   s.db.ToPgxTimestamp(req.Time),
			}); err != nil {
				s.logger.Errorf("failed to store transaction: %v", err)
			}

			if err := s.persistTxStat(ctx, finalizedTx, txhash, req.Height, req.Time); err != nil {
				// don't halt consensus on this
				s.logger.Errorf("failed to persist tx stat: %v", err)
			}

			// set finalized txs in finalize step to remove from mempool during commit step
			// always append to finalized even in error conditions to be removed from mempool
			state.finalizedTxs = append(state.finalizedTxs, txhash)
		} else {
			logger.Errorf("Error: invalid transaction index %v", i)
			txs[i] = &abcitypes.ExecTxResult{Code: 1}
		}
	}

	s.syncPoS(ctx, req.Hash, req.Height)

	nextAppHash := s.serializeAppState([]byte{}, req.GetTxs())

	if err := s.getDb().UpsertAppState(ctx, db.UpsertAppStateParams{
		BlockHeight: req.Height,
		AppHash:     nextAppHash,
	}); err != nil {
		s.logger.Errorf("error upserting app state %v", err)
	}

	// increment number of proposed blocks for sla auditor
	addr := cometbfttypes.Address(req.ProposerAddress).String()
	if err := s.getDb().UpsertSlaRollupReport(ctx, addr); err != nil {
		s.logger.Error(
			"Error attempting to increment blocks proposed by node",
			"address",
			addr,
			"error",
			err,
		)
	}

	// routine every hundredth block to remove expired txs
	// run in separate goroutine to not affect consensus time
	hundredthBlock := req.Height%100 == 0
	if hundredthBlock {
		go s.mempl.RemoveExpiredTransactions(req.Height)
	}

	validatorUpdates := make(abcitypes.ValidatorUpdates, 0, len(validatorUpdatesMap))
	for _, vu := range validatorUpdatesMap {
		validatorUpdates = append(validatorUpdates, vu)
	}

	resp := &abcitypes.FinalizeBlockResponse{
		TxResults: txs,
		AppHash:   nextAppHash,
	}

	if validatorUpdates.Len() > 0 {
		resp.ValidatorUpdates = validatorUpdates
	}

	s.z.Info("block finalized", zap.Int64("height", req.Height), zap.Int("txs", len(req.Txs)))

	return resp, nil
}

func (s *Server) Commit(ctx context.Context, commit *abcitypes.CommitRequest) (*abcitypes.CommitResponse, error) {
	state := s.abciState

	if err := s.commitInProgressTx(ctx); err != nil {
		s.logger.Error("failure to commit tx", "error", err)
		return &abcitypes.CommitResponse{}, err
	}

	// rm txs from mempool
	s.mempl.RemoveBatch(state.finalizedTxs)
	// broadcast txs to subscribers
	for _, txhash := range state.finalizedTxs {
		s.txPubsub.Publish(ctx, txhash, struct{}{})
	}
	// reset abci finalized txs
	state.finalizedTxs = []string{}

	resp := &abcitypes.CommitResponse{}
	if !s.config.Archive {
		latestBlock, err := s.db.GetLatestBlock(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return resp, nil
			}
			s.logger.Errorf("could not get latest block, can't prune: %v", err)
			return resp, nil
		}

		latestBlockHeight := latestBlock.Height
		lastRetainHeight := state.lastRetainHeight
		retainHeight := s.config.RetainHeight

		if latestBlockHeight-retainHeight > lastRetainHeight {
			state.lastRetainHeight = latestBlockHeight
			resp.RetainHeight = state.lastRetainHeight
		}
	}

	return resp, nil
}

func (s *Server) ListSnapshots(_ context.Context, snapshots *abcitypes.ListSnapshotsRequest) (*abcitypes.ListSnapshotsResponse, error) {
	storedSnapshots, err := s.getStoredSnapshots()
	if err != nil {
		return nil, err
	}

	availableSnapshots := make([]*abcitypes.Snapshot, len(storedSnapshots))
	for i, snapshot := range storedSnapshots {
		availableSnapshots[i] = &abcitypes.Snapshot{
			Height:   snapshot.Height,
			Hash:     snapshot.Hash,
			Format:   snapshot.Format,
			Chunks:   snapshot.Chunks,
			Metadata: snapshot.Metadata,
		}
	}

	return &abcitypes.ListSnapshotsResponse{Snapshots: availableSnapshots}, nil
}

func (s *Server) LoadSnapshotChunk(_ context.Context, chunk *abcitypes.LoadSnapshotChunkRequest) (*abcitypes.LoadSnapshotChunkResponse, error) {
	chunkData, err := s.GetChunkByHeight(int64(chunk.Height), int(chunk.Chunk))
	if err != nil {
		s.logger.Error("failed to get chunk", "height", chunk.Height, "chunk", chunk.Chunk, "err", err)
		return nil, err
	}

	return &abcitypes.LoadSnapshotChunkResponse{Chunk: chunkData}, nil
}

func (s *Server) OfferSnapshot(_ context.Context, req *abcitypes.OfferSnapshotRequest) (*abcitypes.OfferSnapshotResponse, error) {
	err := s.StoreOfferedSnapshot(req.Snapshot)
	if err != nil {
		return &abcitypes.OfferSnapshotResponse{
			Result: abcitypes.OFFER_SNAPSHOT_RESULT_REJECT,
		}, nil
	}

	return &abcitypes.OfferSnapshotResponse{
		Result: abcitypes.OFFER_SNAPSHOT_RESULT_ACCEPT,
	}, nil
}

func (s *Server) ApplySnapshotChunk(_ context.Context, req *abcitypes.ApplySnapshotChunkRequest) (*abcitypes.ApplySnapshotChunkResponse, error) {
	offeredSnapshot, err := s.GetOfferedSnapshot()
	if err != nil {
		return &abcitypes.ApplySnapshotChunkResponse{
			Result: abcitypes.APPLY_SNAPSHOT_CHUNK_RESULT_RETRY,
		}, nil
	}

	offeredMetadata := &Metadata{}
	if err := json.Unmarshal(offeredSnapshot.Metadata, offeredMetadata); err != nil {
		return &abcitypes.ApplySnapshotChunkResponse{
			Result: abcitypes.APPLY_SNAPSHOT_CHUNK_RESULT_REJECT_SNAPSHOT,
		}, nil
	}

	if offeredMetadata.ChainID != s.config.GenesisFile.ChainID {
		return &abcitypes.ApplySnapshotChunkResponse{
			Result: abcitypes.APPLY_SNAPSHOT_CHUNK_RESULT_REJECT_SNAPSHOT,
		}, nil
	}

	// if sender is not the same as the offered snapshot, reject
	if !strings.EqualFold(offeredMetadata.Sender, req.Sender) {
		return &abcitypes.ApplySnapshotChunkResponse{
			Result:        abcitypes.APPLY_SNAPSHOT_CHUNK_RESULT_RETRY,
			RejectSenders: []string{req.Sender},
		}, nil
	}

	height := int64(offeredSnapshot.Height)
	totalChunks := int(offeredSnapshot.Chunks)
	chunkIndex := int(req.Index)

	// Store the chunk on disk
	err = s.StoreChunkForReconstruction(height, chunkIndex, req.Chunk)
	if err != nil {
		s.logger.Error("failed to store chunk", "index", chunkIndex, "err", err)
		return &abcitypes.ApplySnapshotChunkResponse{
			Result: abcitypes.APPLY_SNAPSHOT_CHUNK_RESULT_RETRY,
		}, nil
	}

	s.logger.Info("snapshot chunk stored", "height", height, "chunkIndex", chunkIndex, "totalChunks", totalChunks)

	// Check if all chunks are now present on disk
	if s.haveAllChunks(uint64(height), totalChunks) {
		s.logger.Info("all snapshot chunks received, beginning reassembly and restore", "height", height)

		if err := s.ReassemblePgDump(height); err != nil {
			s.logger.Error("failed to reassemble pg_dump", "err", err)
			return &abcitypes.ApplySnapshotChunkResponse{
				Result: abcitypes.APPLY_SNAPSHOT_CHUNK_RESULT_RETRY,
			}, nil
		}

		if err := s.RestoreDatabase(height); err != nil {
			s.logger.Error("failed to restore database", "err", err)
			return &abcitypes.ApplySnapshotChunkResponse{
				Result: abcitypes.APPLY_SNAPSHOT_CHUNK_RESULT_RETRY,
			}, nil
		}

		if err := s.CleanupStateSync(); err != nil {
			// don't need to fail the snapshot chunk if cleanup fails
			s.logger.Warn("failed to cleanup state sync", "err", err)
		}

		return &abcitypes.ApplySnapshotChunkResponse{
			Result: abcitypes.APPLY_SNAPSHOT_CHUNK_RESULT_ACCEPT,
		}, nil
	}

	return &abcitypes.ApplySnapshotChunkResponse{
		Result: abcitypes.APPLY_SNAPSHOT_CHUNK_RESULT_ACCEPT,
	}, nil
}

func (s *Server) ExtendVote(_ context.Context, extend *abcitypes.ExtendVoteRequest) (*abcitypes.ExtendVoteResponse, error) {
	return &abcitypes.ExtendVoteResponse{}, nil
}

func (s *Server) VerifyVoteExtension(_ context.Context, verify *abcitypes.VerifyVoteExtensionRequest) (*abcitypes.VerifyVoteExtensionResponse, error) {
	return &abcitypes.VerifyVoteExtensionResponse{}, nil
}

//////////////////////////////////
//// Utility Methods for ABCI ////
//////////////////////////////////

// returns in current postgres tx for this block
func (s *Server) getDb() *db.Queries {
	return s.db.WithTx(s.abciState.onGoingBlock)
}

func (s *Server) startInProgressTx(ctx context.Context) error {
	dbTx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}

	s.abciState.onGoingBlock = dbTx
	return nil
}

// commits the current tx that's finished indexing
func (s *Server) commitInProgressTx(ctx context.Context) error {
	state := s.abciState
	if state.onGoingBlock != nil {
		err := state.onGoingBlock.Commit(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrTxClosed) {
				state.onGoingBlock = nil
				return nil
			}
			return err
		}
		state.onGoingBlock = nil
	}
	return nil
}

func (s *Server) isValidSignedTransaction(tx []byte) (*v1.SignedTransaction, error) {
	var msg v1.SignedTransaction
	err := proto.Unmarshal(tx, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *Server) validateBlockTxs(ctx context.Context, blockTime time.Time, blockHeight int64, misbehavior []abcitypes.Misbehavior, txs [][]byte) (bool, error) {
	for _, tx := range txs {
		valid, err := s.validateBlockTx(ctx, blockTime, blockHeight, misbehavior, tx)
		if err != nil {
			return false, err
		} else if !valid {
			return false, nil
		}
	}
	return true, nil
}

func (s *Server) validateBlockTx(ctx context.Context, blockTime time.Time, blockHeight int64, misbehavior []abcitypes.Misbehavior, tx []byte) (bool, error) {
	signedTx, err := s.isValidSignedTransaction(tx)
	if err != nil {
		s.logger.Error("Invalid block: unrecognized transaction type")
		return false, nil
	}

	switch signedTx.Transaction.(type) {
	case *v1.SignedTransaction_Plays:
	case *v1.SignedTransaction_Attestation:
		if err := s.isValidAttestation(ctx, signedTx, blockHeight); err != nil {
			s.logger.Error("Invalid block: invalid attestation tx", "error", err)
			return false, nil
		}
	case *v1.SignedTransaction_ValidatorDeregistration:
		if err := s.isValidDeregisterMisbehavingNodeTx(signedTx, misbehavior); err != nil {
			s.logger.Error("Invalid block: invalid deregister node tx", "error", err)
			return false, nil
		}
	case *v1.SignedTransaction_SlaRollup:
		if valid, err := s.isValidRollup(ctx, blockTime, blockHeight, signedTx.GetSlaRollup()); err != nil {
			s.logger.Error("Invalid block: error validating sla rollup", "error", err)
			return false, err
		} else if !valid {
			s.logger.Error("Invalid block: invalid rollup")
			return false, nil
		}
	case *v1.SignedTransaction_StorageProof:
		if err := s.isValidStorageProofTx(ctx, signedTx, blockHeight, true); err != nil {
			s.logger.Error("Invalid block: invalid storage proof tx", "error", err)
			return false, nil
		}
	case *v1.SignedTransaction_StorageProofVerification:
		if err := s.isValidStorageProofVerificationTx(ctx, signedTx, blockHeight); err != nil {
			s.logger.Error("Invalid block: invalid storage proof verification tx", "error", err)
			return false, nil
		}
	case *v1.SignedTransaction_Release:
		if err := s.isValidReleaseTx(ctx, signedTx); err != nil {
			s.logger.Error("Invalid block: invalid release tx", "error", err)
			return false, nil
		}
	}
	return true, nil
}

func (s *Server) finalizeTransaction(ctx context.Context, req *abcitypes.FinalizeBlockRequest, msg *v1.SignedTransaction, txHash string, blockHeight int64) (proto.Message, error) {
	misbehavior := req.Misbehavior
	switch t := msg.Transaction.(type) {
	case *v1.SignedTransaction_Plays:
		return s.finalizePlayTransaction(ctx, msg)
	case *v1.SignedTransaction_ManageEntity:
		return s.finalizeManageEntity(ctx, msg)
	case *v1.SignedTransaction_Attestation:
		return s.finalizeAttestation(ctx, msg, req.Height)
	case *v1.SignedTransaction_ValidatorRegistration:
		return s.finalizeLegacyRegisterNode(ctx, msg, blockHeight)
	case *v1.SignedTransaction_ValidatorDeregistration:
		return s.finalizeDeregisterMisbehavingNode(ctx, msg, misbehavior)
	case *v1.SignedTransaction_SlaRollup:
		return s.finalizeSlaRollup(ctx, msg, txHash)
	case *v1.SignedTransaction_StorageProof:
		return s.finalizeStorageProof(ctx, msg, blockHeight)
	case *v1.SignedTransaction_StorageProofVerification:
		return s.finalizeStorageProofVerification(ctx, msg, blockHeight)
	case *v1.SignedTransaction_Release:
		return s.finalizeRelease(ctx, msg, txHash)
	default:
		return nil, fmt.Errorf("unhandled proto event: %v %T", msg, t)
	}
}

func (s *Server) persistTxStat(ctx context.Context, tx proto.Message, txhash string, height int64, blockTime time.Time) error {
	if tx == nil {
		return nil
	}
	if err := s.getDb().InsertTxStat(ctx, db.InsertTxStatParams{
		TxType:      GetProtoTypeName(tx),
		TxHash:      txhash,
		BlockHeight: height,
		CreatedAt: pgtype.Timestamp{
			Time:  blockTime,
			Valid: true,
		},
	}); err != nil {
		s.logger.Error("error inserting tx stat", "error", err)
	}
	return nil
}

func (s *Server) serializeAppState(prevHash []byte, txs [][]byte) []byte {
	var combinedHash []byte

	combinedHash = append(combinedHash, prevHash...)

	for _, tx := range txs {
		combinedHash = append(combinedHash, tx...)
	}

	newAppHashBytes := sha256.Sum256(combinedHash)
	return newAppHashBytes[:]
}

func (s *Server) toTxHash(msg proto.Message) string {
	hash, err := common.ToTxHash(msg)
	if err != nil {
		s.logger.Errorf("could not get txhash of msg: %v %v", msg, err)
		return ""
	}
	return hash
}
