package server

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"connectrpc.com/connect"
	"dario.cat/mergo"
	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/api/core/v1/v1connect"
	v1beta1 "github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
	"github.com/AudiusProject/audiusd/pkg/api/ddex/v1beta2"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/rewards"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CoreService struct {
	core *Server
}

func NewCoreService() *CoreService {
	return &CoreService{}
}

func (c *CoreService) SetCore(core *Server) {
	c.core = core
	c.core.setSelf(c)
}

var _ v1connect.CoreServiceHandler = (*CoreService)(nil)

// GetNodeInfo implements v1connect.CoreServiceHandler.
func (c *CoreService) GetNodeInfo(ctx context.Context, req *connect.Request[v1.GetNodeInfoRequest]) (*connect.Response[v1.GetNodeInfoResponse], error) {
	status, err := c.core.rpc.Status(ctx)
	if err != nil {
		return nil, err
	}

	res := &v1.GetNodeInfoResponse{
		Chainid:       c.core.config.GenesisFile.ChainID,
		Synced:        !status.SyncInfo.CatchingUp,
		CometAddress:  c.core.config.ProposerAddress,
		EthAddress:    c.core.config.WalletAddress,
		CurrentHeight: status.SyncInfo.LatestBlockHeight,
	}
	return connect.NewResponse(res), nil
}

// ForwardTransaction implements v1connect.CoreServiceHandler.
func (c *CoreService) ForwardTransaction(ctx context.Context, req *connect.Request[v1.ForwardTransactionRequest]) (*connect.Response[v1.ForwardTransactionResponse], error) {
	// TODO: check signature from known node

	// TODO: validate transaction in same way as send transaction

	var mempoolKey common.TxHash
	var err error
	if req.Msg.Transactionv2 != nil {
		mempoolKey, err = common.ToTxHash(req.Msg.Transactionv2.Envelope)
	} else {
		mempoolKey, err = common.ToTxHash(req.Msg.Transaction)
	}
	if err != nil {
		return nil, fmt.Errorf("could not get tx hash of signed tx: %v", err)
	}

	if req.Msg.Transactionv2 != nil {
		c.core.logger.Debugf("received forwarded v2 tx: %v", req.Msg.Transactionv2)
	} else {
		c.core.logger.Debugf("received forwarded tx: %v", req.Msg.Transaction)
	}

	// TODO: intake block deadline from request
	status, err := c.core.rpc.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("chain not healthy: %v", err)
	}

	deadline := status.SyncInfo.LatestBlockHeight + 10
	var mempoolTx *MempoolTransaction
	if req.Msg.Transaction != nil {
		mempoolTx = &MempoolTransaction{
			Tx:       req.Msg.Transaction,
			Deadline: deadline,
		}
	} else if req.Msg.Transactionv2 != nil {
		mempoolTx = &MempoolTransaction{
			Txv2:     req.Msg.Transactionv2,
			Deadline: deadline,
		}
	} else {
		return nil, fmt.Errorf("no transaction provided")
	}

	err = c.core.addMempoolTransaction(mempoolKey, mempoolTx, false)
	if err != nil {
		return nil, fmt.Errorf("could not add tx to mempool %v", err)
	}

	return connect.NewResponse(&v1.ForwardTransactionResponse{}), nil
}

// GetBlock implements v1connect.CoreServiceHandler.
func (c *CoreService) GetBlock(ctx context.Context, req *connect.Request[v1.GetBlockRequest]) (*connect.Response[v1.GetBlockResponse], error) {
	currentHeight := c.core.cache.currentHeight.Load()
	if req.Msg.Height > currentHeight {
		return connect.NewResponse(&v1.GetBlockResponse{
			Block: &v1.Block{
				ChainId: c.core.config.GenesisFile.ChainID,
				Height:  -1,
			},
		}), nil
	}

	block, err := c.core.db.GetBlock(ctx, req.Msg.Height)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// fallback to rpc for now, remove after mainnet-alpha
			return c.getBlockRpcFallback(ctx, req.Msg.Height)
		}
		c.core.logger.Errorf("error getting block: %v", err)
		return nil, err
	}

	blockTxs, err := c.core.db.GetBlockTransactions(ctx, req.Msg.Height)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	txResponses := []*v1.Transaction{}
	for _, tx := range blockTxs {
		var transaction v1.SignedTransaction
		err = proto.Unmarshal(tx.Transaction, &transaction)
		if err != nil {
			return nil, err
		}
		res := &v1.Transaction{
			Hash:        tx.TxHash,
			BlockHash:   block.Hash,
			ChainId:     c.core.config.GenesisFile.ChainID,
			Height:      block.Height,
			Timestamp:   timestamppb.New(block.CreatedAt.Time),
			Transaction: &transaction,
		}
		txResponses = append(txResponses, res)
	}

	res := &v1.Block{
		Hash:         block.Hash,
		ChainId:      c.core.config.GenesisFile.ChainID,
		Proposer:     block.Proposer,
		Height:       block.Height,
		Transactions: sortTransactionResponse(txResponses),
		Timestamp:    timestamppb.New(block.CreatedAt.Time),
	}

	return connect.NewResponse(&v1.GetBlockResponse{Block: res, CurrentHeight: c.core.cache.currentHeight.Load()}), nil
}

// GetDeregistrationAttestation implements v1connect.CoreServiceHandler.
func (c *CoreService) GetDeregistrationAttestation(ctx context.Context, req *connect.Request[v1.GetDeregistrationAttestationRequest]) (*connect.Response[v1.GetDeregistrationAttestationResponse], error) {
	dereg := req.Msg.Deregistration
	if dereg == nil {
		return nil, errors.New("empty deregistration attestation")
	}

	node, err := c.core.db.GetRegisteredNodeByCometAddress(ctx, dereg.CometAddress)
	if err != nil {
		return nil, fmt.Errorf("could not attest deregistration for '%s': %v", dereg.CometAddress, err)
	}

	ethBlock := new(big.Int)
	ethBlock, ok := ethBlock.SetString(node.EthBlock, 10)
	if !ok {
		return nil, fmt.Errorf("could not format eth block '%s' for node '%s'", node.EthBlock, node.Endpoint)
	}

	if registered, err := c.core.IsNodeRegisteredOnEthereum(
		ctx,
		node.Endpoint,
		node.EthAddress,
		ethBlock.Int64(),
	); registered || err != nil {
		c.core.logger.Error("Could not attest to node eth deregistration: node is still registered",
			"cometAddress",
			dereg.CometAddress,
			"ethAddress",
			node.EthAddress,
			"endpoint",
			node.Endpoint,
			"error",
			err,
		)
		return nil, errors.New("node is still registered on ethereum")
	}

	deregBytes, err := proto.Marshal(dereg)
	if err != nil {
		c.core.logger.Error("could not marshal deregistration", "error", err)
		return nil, err
	}
	sig, err := common.EthSign(c.core.config.EthereumKey, deregBytes)
	if err != nil {
		c.core.logger.Error("could not sign deregistration", "error", err)
		return nil, err
	}

	return connect.NewResponse(&v1.GetDeregistrationAttestationResponse{
		Signature:      sig,
		Deregistration: dereg,
	}), nil
}

// GetHealth implements v1connect.CoreServiceHandler.
func (c *CoreService) GetHealth(context.Context, *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error) {
	return connect.NewResponse(&v1.GetHealthResponse{}), nil
}

// GetRegistrationAttestation implements v1connect.CoreServiceHandler.
func (c *CoreService) GetRegistrationAttestation(ctx context.Context, req *connect.Request[v1.GetRegistrationAttestationRequest]) (*connect.Response[v1.GetRegistrationAttestationResponse], error) {
	reg := req.Msg.Registration
	if reg == nil {
		return nil, errors.New("empty registration attestation")
	}

	if reg.Deadline < c.core.cache.currentHeight.Load() || reg.Deadline > c.core.cache.currentHeight.Load()+maxRegistrationAttestationValidity {
		return nil, fmt.Errorf("cannot sign registration request with deadline %d (current height is %d)", reg.Deadline, c.core.cache.currentHeight.Load())
	}

	if registered, err := c.core.IsNodeRegisteredOnEthereum(
		ctx,
		reg.Endpoint,
		reg.DelegateWallet,
		reg.EthBlock,
	); !registered || err != nil {
		c.core.logger.Error(
			"Could not attest to node eth registration",
			"delegate",
			reg.DelegateWallet,
			"endpoint",
			reg.Endpoint,
			"eth block",
			reg.EthBlock,
			"error",
			err,
		)
		return nil, errors.New("node is not registered on ethereum")
	}

	regBytes, err := proto.Marshal(reg)
	if err != nil {
		c.core.logger.Error("could not marshal registration", "error", err)
		return nil, err
	}
	sig, err := common.EthSign(c.core.config.EthereumKey, regBytes)
	if err != nil {
		c.core.logger.Error("could not sign registration", "error", err)
		return nil, err
	}

	return connect.NewResponse(&v1.GetRegistrationAttestationResponse{
		Signature:    sig,
		Registration: reg,
	}), nil
}

// GetTransaction implements v1connect.CoreServiceHandler.
func (c *CoreService) GetTransaction(ctx context.Context, req *connect.Request[v1.GetTransactionRequest]) (*connect.Response[v1.GetTransactionResponse], error) {
	txhash := req.Msg.TxHash

	c.core.logger.Debug("query", "txhash", txhash)

	tx, err := c.core.db.GetTx(ctx, txhash)
	if err != nil {
		return nil, err
	}

	block, err := c.core.db.GetBlock(ctx, tx.BlockID)
	if err != nil {
		return nil, err
	}

	// Try to unmarshal as v1 transaction first
	var v1Transaction v1.SignedTransaction
	err = proto.Unmarshal(tx.Transaction, &v1Transaction)
	if err == nil {
		// Successfully unmarshaled as v1 transaction
		return connect.NewResponse(&v1.GetTransactionResponse{
			Transaction: &v1.Transaction{
				Hash:        txhash,
				BlockHash:   block.Hash,
				ChainId:     c.core.config.GenesisFile.ChainID,
				Height:      block.Height,
				Timestamp:   timestamppb.New(block.CreatedAt.Time),
				Transaction: &v1Transaction,
			},
		}), nil
	}

	// Try to unmarshal as v2 transaction
	var v2Transaction v1beta1.Transaction
	err = proto.Unmarshal(tx.Transaction, &v2Transaction)
	if err == nil {
		// Successfully unmarshaled as v2 transaction
		// For now, return the v2 transaction in the response - the API might need to be extended
		// to properly handle v2 transactions, but this allows retrieval without error
		return connect.NewResponse(&v1.GetTransactionResponse{
			Transaction: &v1.Transaction{
				Hash:          txhash,
				BlockHash:     block.Hash,
				ChainId:       c.core.config.GenesisFile.ChainID,
				Height:        block.Height,
				Timestamp:     timestamppb.New(block.CreatedAt.Time),
				Transaction:   &v1Transaction,
				Transactionv2: &v2Transaction,
			},
		}), nil
	}

	// If neither worked, return the original error
	return nil, fmt.Errorf("could not unmarshal transaction as v1 or v2: %v", err)
}

// Ping implements v1connect.CoreServiceHandler.
func (c *CoreService) Ping(context.Context, *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Message: "pong"}), nil
}

// SendTransaction implements v1connect.CoreServiceHandler.
func (c *CoreService) SendTransaction(ctx context.Context, req *connect.Request[v1.SendTransactionRequest]) (*connect.Response[v1.SendTransactionResponse], error) {
	// TODO: do validation check
	var txhash common.TxHash
	var err error
	if req.Msg.Transactionv2 != nil {
		txhash, err = common.ToTxHash(req.Msg.Transactionv2.Envelope)
	} else {
		txhash, err = common.ToTxHash(req.Msg.Transaction)
	}
	if err != nil {
		return nil, fmt.Errorf("could not get tx hash of signed tx: %v", err)
	}

	// create mempool transaction for both v1 and v2
	var mempoolTx *MempoolTransaction
	deadline := c.core.cache.currentHeight.Load() + 10
	if req.Msg.Transaction != nil {
		mempoolTx = &MempoolTransaction{
			Tx:       req.Msg.Transaction,
			Deadline: deadline,
		}
	} else if req.Msg.Transactionv2 != nil {
		mempoolTx = &MempoolTransaction{
			Txv2:     req.Msg.Transactionv2,
			Deadline: deadline,
		}
	}

	ps := c.core.txPubsub

	txHashCh := ps.Subscribe(txhash)
	defer ps.Unsubscribe(txhash, txHashCh)

	// add transaction to mempool with broadcast set to true
	if mempoolTx != nil {
		err = c.core.addMempoolTransaction(txhash, mempoolTx, true)
		if err != nil {
			c.core.logger.Errorf("tx could not be included in mempool %s: %v", txhash, err)
			return nil, fmt.Errorf("could not add tx to mempool %v", err)
		}
	}

	select {
	case <-txHashCh:
		tx, err := c.core.db.GetTx(ctx, txhash)
		if err != nil {
			return nil, err
		}

		block, err := c.core.db.GetBlock(ctx, tx.BlockID)
		if err != nil {
			return nil, err
		}

		return connect.NewResponse(&v1.SendTransactionResponse{
			Transaction: &v1.Transaction{
				Hash:        txhash,
				BlockHash:   block.Hash,
				ChainId:     c.core.config.GenesisFile.ChainID,
				Height:      block.Height,
				Timestamp:   timestamppb.New(block.CreatedAt.Time),
				Transaction: req.Msg.Transaction,
			},
		}), nil
	case <-time.After(30 * time.Second):
		c.core.logger.Errorf("tx timeout waiting to be included %s", txhash)
		return nil, errors.New("tx waiting timeout")
	}
}

// Utilities
func (c *CoreService) getBlockRpcFallback(ctx context.Context, height int64) (*connect.Response[v1.GetBlockResponse], error) {
	block, err := c.core.rpc.Block(ctx, &height)
	if err != nil {
		blockInFutureMsg := "must be less than or equal to the current blockchain height"
		if strings.Contains(err.Error(), blockInFutureMsg) {
			// return block with -1 to indicate it doesn't exist yet
			return connect.NewResponse(&v1.GetBlockResponse{
				Block: &v1.Block{
					ChainId:   c.core.config.GenesisFile.ChainID,
					Height:    -1,
					Timestamp: timestamppb.New(time.Now()),
				},
			}), nil
		}
		c.core.logger.Errorf("error getting block: %v", err)
		return nil, err
	}

	txs := []*v1.Transaction{}
	for _, tx := range block.Block.Txs {
		var transaction v1.SignedTransaction
		err = proto.Unmarshal(tx, &transaction)
		if err != nil {
			return nil, err
		}
		txs = append(txs, &v1.Transaction{
			Hash:        c.core.toTxHash(&transaction),
			BlockHash:   block.BlockID.Hash.String(),
			ChainId:     c.core.config.GenesisFile.ChainID,
			Height:      block.Block.Height,
			Timestamp:   timestamppb.New(block.Block.Time),
			Transaction: &transaction,
		})
	}

	txs = sortTransactionResponse(txs)

	res := &v1.GetBlockResponse{
		Block: &v1.Block{
			Hash:         block.BlockID.Hash.String(),
			ChainId:      c.core.config.GenesisFile.ChainID,
			Proposer:     block.Block.ProposerAddress.String(),
			Height:       block.Block.Height,
			Transactions: txs,
			Timestamp:    timestamppb.New(block.Block.Time),
		},
	}

	return connect.NewResponse(res), nil
}

// GetStoredSnapshots implements v1connect.CoreServiceHandler.
func (c *CoreService) GetStoredSnapshots(context.Context, *connect.Request[v1.GetStoredSnapshotsRequest]) (*connect.Response[v1.GetStoredSnapshotsResponse], error) {
	snapshots, err := c.core.getStoredSnapshots()
	if err != nil {
		c.core.logger.Errorf("error getting stored snapshots: %v", err)
		return connect.NewResponse(&v1.GetStoredSnapshotsResponse{
			Snapshots: []*v1.SnapshotMetadata{},
		}), nil
	}

	snapshotResponses := make([]*v1.SnapshotMetadata, 0, len(snapshots))
	for _, snapshot := range snapshots {
		snapshotResponses = append(snapshotResponses, &v1.SnapshotMetadata{
			Height:     int64(snapshot.Height),
			Hash:       hex.EncodeToString(snapshot.Hash),
			ChunkCount: int64(snapshot.Chunks),
			ChainId:    string(snapshot.Metadata),
		})
	}

	res := &v1.GetStoredSnapshotsResponse{
		Snapshots: snapshotResponses,
	}

	return connect.NewResponse(res), nil
}

// GetStatus implements v1connect.CoreServiceHandler.
func (c *CoreService) GetStatus(context.Context, *connect.Request[v1.GetStatusRequest]) (*connect.Response[v1.GetStatusResponse], error) {
	live := true
	ready := false

	res := &v1.GetStatusResponse{
		Live:  live,
		Ready: ready,
	}

	if c.core == nil {
		return connect.NewResponse(res), nil
	}

	nodeInfo, _ := c.core.cache.nodeInfo.Get(NodeInfoKey)
	peers, _ := c.core.cache.peers.Get(PeersKey)
	chainInfo, _ := c.core.cache.chainInfo.Get(ChainInfoKey)
	syncInfo, _ := c.core.cache.syncInfo.Get(SyncInfoKey)
	pruningInfo, _ := c.core.cache.pruningInfo.Get(PruningInfoKey)
	resourceInfo, _ := c.core.cache.resourceInfo.Get(ResourceInfoKey)
	mempoolInfo, _ := c.core.cache.mempoolInfo.Get(MempoolInfoKey)
	snapshotInfo, _ := c.core.cache.snapshotInfo.Get(SnapshotInfoKey)

	peersOk := len(peers.P2P) > 0 && len(peers.Rpc) > 0
	syncInfoOk := syncInfo.Synced
	diskOk := resourceInfo.DiskFree > 0
	memOk := resourceInfo.MemUsage < resourceInfo.MemSize
	cpuOk := resourceInfo.CpuUsage < 100
	ready = peersOk && syncInfoOk && diskOk && memOk && cpuOk

	res.Ready = ready
	res.NodeInfo = nodeInfo
	res.Peers = peers
	res.ChainInfo = chainInfo
	res.SyncInfo = syncInfo
	res.PruningInfo = pruningInfo
	res.ResourceInfo = resourceInfo
	res.MempoolInfo = mempoolInfo
	res.SnapshotInfo = snapshotInfo

	return connect.NewResponse(res), nil
}

// GetRewardAttestation implements v1connect.CoreServiceHandler.
func (c *CoreService) GetRewardAttestation(ctx context.Context, req *connect.Request[v1.GetRewardAttestationRequest]) (*connect.Response[v1.GetRewardAttestationResponse], error) {
	ethRecipientAddress := req.Msg.EthRecipientAddress
	if ethRecipientAddress == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("eth_recipient_address is required"))
	}
	rewardID := req.Msg.RewardId
	if rewardID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("reward_id is required"))
	}
	specifier := req.Msg.Specifier
	if specifier == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("specifier is required"))
	}
	oracleAddress := req.Msg.OracleAddress
	if oracleAddress == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("oracle_address is required"))
	}
	signature := req.Msg.Signature
	if signature == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("signature is required"))
	}
	amount := req.Msg.Amount
	if amount == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("amount is required"))
	}

	claim := rewards.RewardClaim{
		RecipientEthAddress:       ethRecipientAddress,
		Amount:                    amount,
		RewardID:                  rewardID,
		Specifier:                 specifier,
		AntiAbuseOracleEthAddress: oracleAddress,
	}

	err := c.core.rewards.Validate(claim)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = c.core.rewards.Authenticate(claim, signature)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	_, attestation, err := c.core.rewards.Attest(claim)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &v1.GetRewardAttestationResponse{
		Owner:       c.core.rewards.EthereumAddress,
		Attestation: attestation,
	}

	return connect.NewResponse(res), nil
}

// GetRewards implements v1connect.CoreServiceHandler.
func (c *CoreService) GetRewards(context.Context, *connect.Request[v1.GetRewardsRequest]) (*connect.Response[v1.GetRewardsResponse], error) {
	rewards := c.core.rewards.Rewards
	rewardResponses := make([]*v1.Reward, 0, len(rewards))
	for _, reward := range rewards {
		claimAuthorities := make([]*v1.ClaimAuthority, 0, len(reward.ClaimAuthorities))
		for _, claimAuthority := range reward.ClaimAuthorities {
			claimAuthorities = append(claimAuthorities, &v1.ClaimAuthority{
				Address: claimAuthority.Address,
				Name:    claimAuthority.Name,
			})
		}
		rewardResponses = append(rewardResponses, &v1.Reward{
			RewardId:         reward.RewardId,
			Amount:           reward.Amount,
			Name:             reward.Name,
			ClaimAuthorities: claimAuthorities,
		})
	}

	res := &v1.GetRewardsResponse{
		Rewards: rewardResponses,
	}

	return connect.NewResponse(res), nil
}

// GetERN implements v1connect.CoreServiceHandler.
func (c *CoreService) GetERN(ctx context.Context, req *connect.Request[v1.GetERNRequest]) (*connect.Response[v1.GetERNResponse], error) {
	ernMessages, err := c.core.db.GetERNMessages(ctx, req.Msg.Address)
	if err != nil {
		return nil, err
	}

	if len(ernMessages) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("ERN message not found"))
	}

	// TODO: make merging generic
	// Unmarshal the first message as the base
	baseERN := &v1beta2.NewReleaseMessage{}
	err = proto.Unmarshal(ernMessages[0].RawErnMessage, baseERN)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal base ERN message: %w", err)
	}

	// Merge all subsequent ERN messages into the base using mergo
	for i := 1; i < len(ernMessages); i++ {
		currentERN := &v1beta2.NewReleaseMessage{}
		err = proto.Unmarshal(ernMessages[i].RawErnMessage, currentERN)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal ERN message at index %d: %w", i, err)
		}

		// Merge current ERN into base ERN
		// WithOverride ensures newer values take precedence
		// WithAppendSlice appends slices instead of replacing them
		// WithoutDereference prevents copying the protobuf internal state
		err = mergo.Merge(baseERN, currentERN,
			mergo.WithOverride,
			mergo.WithAppendSlice,
			mergo.WithoutDereference,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to merge ERN message at index %d: %w", i, err)
		}
	}

	return connect.NewResponse(&v1.GetERNResponse{Ern: baseERN}), nil
}
