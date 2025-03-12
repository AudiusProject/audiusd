package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	cfg "github.com/cometbft/cometbft/config"
	cmtflags "github.com/cometbft/cometbft/libs/cli/flags"
	nm "github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"
)

const (
	week = 7 * 24 * time.Hour
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
func (s *Server) startABCI() error {
	<-s.awaitEthNodesReady
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
		context.Background(),
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

	s.logger.Info("said node was ready")

	s.rpc = local.New(s.node)
	close(s.awaitRpcReady)

	s.logger.Info("core CometBFT node starting")

	if err := s.node.Start(); err != nil {
		s.logger.Errorf("cometbft failed to start: %v", err)
		return err
	}
	return nil
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
	var txs = make([]*abcitypes.ExecTxResult, len(req.Txs))

	// Start a new block transaction
	if err := s.startInProgressTx(ctx); err != nil {
		logger.Errorf("failed to start block transaction: %v", err)
		// Continue without transaction - better than halting the chain
	}

	// Store block info - if this fails, log but continue
	if err := s.getDb().StoreBlock(ctx, db.StoreBlockParams{
		Height:    req.Height,
		Hash:      hex.EncodeToString(req.Hash),
		Proposer:  hex.EncodeToString(req.ProposerAddress),
		ChainID:   s.config.GenesisFile.ChainID,
		CreatedAt: s.db.ToPgxTimestamp(req.Time),
	}); err != nil {
		s.handleTxError(ctx, err, "", "could not store block")
	}

	for i, tx := range req.Txs {
		// Set default success response
		txs[i] = &abcitypes.ExecTxResult{Code: abcitypes.CodeTypeOK}

		signedTx, err := s.isValidSignedTransaction(tx)
		if err != nil {
			txs[i].Code = 1
			continue
		}

		txhash := s.toTxHash(signedTx)
		finalizedTx, err := s.finalizeTransaction(ctx, req, signedTx, txhash, req.Height)
		if err != nil {
			s.handleTxError(ctx, err, txhash, "failed to finalize transaction")
			txs[i].Code = 1
			continue
		}

		// Store transaction - if this fails, log but continue
		if err := s.persistTxStat(ctx, finalizedTx, txhash, req.Height, req.Time); err != nil {
			s.handleTxError(ctx, err, txhash, "failed to persist transaction stats")
		}
	}

	// Try to commit the transaction block
	if err := s.commitInProgressTx(ctx); err != nil {
		logger.Errorf("failed to commit block transaction: %v", err)
		// Don't return error - better to continue than halt
	}

	// Calculate new app hash regardless of DB errors
	prevState, _ := s.db.GetLatestAppState(ctx)
	prevHash := prevState.AppHash
	newAppHash := s.serializeAppState(prevHash, req.Txs)

	return &abcitypes.FinalizeBlockResponse{
		TxResults: txs,
		AppHash:   newAppHash,
	}, nil
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
	return &abcitypes.ListSnapshotsResponse{}, nil
}

func (s *Server) OfferSnapshot(_ context.Context, snapshot *abcitypes.OfferSnapshotRequest) (*abcitypes.OfferSnapshotResponse, error) {
	return &abcitypes.OfferSnapshotResponse{}, nil
}

func (s *Server) LoadSnapshotChunk(_ context.Context, chunk *abcitypes.LoadSnapshotChunkRequest) (*abcitypes.LoadSnapshotChunkResponse, error) {
	return &abcitypes.LoadSnapshotChunkResponse{}, nil
}

func (s *Server) ApplySnapshotChunk(_ context.Context, chunk *abcitypes.ApplySnapshotChunkRequest) (*abcitypes.ApplySnapshotChunkResponse, error) {
	return &abcitypes.ApplySnapshotChunkResponse{Result: abcitypes.APPLY_SNAPSHOT_CHUNK_RESULT_ACCEPT}, nil
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

func (s *Server) isValidSignedTransaction(tx []byte) (*core_proto.SignedTransaction, error) {
	var msg core_proto.SignedTransaction
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
	case *core_proto.SignedTransaction_Plays:
	case *core_proto.SignedTransaction_ValidatorRegistration:
		if err := s.isValidRegisterNodeTx(signedTx); err != nil {
			s.logger.Error("Invalid block: invalid register node tx", "error", err)
			return false, nil
		}
	case *core_proto.SignedTransaction_ValidatorDeregistration:
		if err := s.isValidDeregisterNodeTx(signedTx, misbehavior); err != nil {
			s.logger.Error("Invalid block: invalid deregister node tx", "error", err)
			return false, nil
		}
	case *core_proto.SignedTransaction_SlaRollup:
		if valid, err := s.isValidRollup(ctx, blockTime, blockHeight, signedTx.GetSlaRollup()); err != nil {
			s.logger.Error("Invalid block: error validating sla rollup", "error", err)
			return false, err
		} else if !valid {
			s.logger.Error("Invalid block: invalid rollup")
			return false, nil
		}
	case *core_proto.SignedTransaction_StorageProof:
		if err := s.isValidStorageProofTx(ctx, signedTx, blockHeight, true); err != nil {
			s.logger.Error("Invalid block: invalid storage proof tx", "error", err)
			return false, nil
		}
	case *core_proto.SignedTransaction_StorageProofVerification:
		if err := s.isValidStorageProofVerificationTx(ctx, signedTx, blockHeight); err != nil {
			s.logger.Error("Invalid block: invalid storage proof verification tx", "error", err)
			return false, nil
		}
	}
	return true, nil
}

func (s *Server) finalizeTransaction(ctx context.Context, req *abcitypes.FinalizeBlockRequest, msg *core_proto.SignedTransaction, txHash string, blockHeight int64) (proto.Message, error) {
	misbehavior := req.Misbehavior
	switch t := msg.Transaction.(type) {
	case *core_proto.SignedTransaction_Plays:
		return s.finalizePlayTransaction(ctx, msg)
	case *core_proto.SignedTransaction_ManageEntity:
		return s.finalizeManageEntity(ctx, msg)
	case *core_proto.SignedTransaction_ValidatorRegistration:
		return s.finalizeRegisterNode(ctx, msg, req.Time)
	case *core_proto.SignedTransaction_ValidatorDeregistration:
		return s.finalizeDeregisterNode(ctx, msg, misbehavior)
	case *core_proto.SignedTransaction_SlaRollup:
		return s.finalizeSlaRollup(ctx, msg, txHash)
	case *core_proto.SignedTransaction_StorageProof:
		return s.finalizeStorageProof(ctx, msg, blockHeight)
	case *core_proto.SignedTransaction_StorageProofVerification:
		return s.finalizeStorageProofVerification(ctx, msg, blockHeight)
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

func (s *Server) handleTxError(ctx context.Context, err error, txHash string, msg string) {
	if err == nil {
		return
	}

	// Log the error
	s.logger.Errorf("%s: %v", msg, err)

	// If it's a transaction that's already been rolled back, don't try to roll back again
	if errors.Is(err, pgx.ErrTxClosed) {
		return
	}

	// Start a new transaction if we don't have one
	if s.abciState.onGoingBlock == nil {
		if err := s.startInProgressTx(ctx); err != nil {
			s.logger.Errorf("failed to start new transaction after error: %v", err)
			return
		}
	}

	// Record the failed transaction
	if err := s.getDb().InsertTxStat(ctx, db.InsertTxStatParams{
		TxType:      "failed_tx",
		TxHash:      txHash,
		BlockHeight: -1, // Indicates failed transaction
		CreatedAt: pgtype.Timestamp{
			Time:  time.Now(),
			Valid: true,
		},
	}); err != nil {
		s.logger.Errorf("failed to record failed transaction: %v", err)
	}
}
