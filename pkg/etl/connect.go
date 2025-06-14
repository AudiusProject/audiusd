package etl

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	corev1connect "github.com/AudiusProject/audiusd/pkg/api/core/v1/v1connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/etl/v1"
	"github.com/AudiusProject/audiusd/pkg/api/etl/v1/v1connect"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/etl/db"
	"github.com/AudiusProject/audiusd/pkg/etl/location"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maypok86/otter"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ v1connect.ETLServiceHandler = (*ETLService)(nil)

type ETLService struct {
	dbURL               string
	runDownMigrations   bool
	startingBlockHeight int64
	endingBlockHeight   int64
	checkReadiness      bool
	chainID             string

	stats otter.Cache[string, Stats]

	core   corev1connect.CoreServiceClient
	pool   *pgxpool.Pool
	db     *db.Queries
	logger *common.Logger

	locationDB *location.LocationService

	blockPubsub *BlockPubsub
	playPubsub  *PlayPubsub
}

func (e *ETLService) SetDBURL(dbURL string) {
	e.dbURL = dbURL
}

func (e *ETLService) SetStartingBlockHeight(startingBlockHeight int64) {
	e.startingBlockHeight = startingBlockHeight
}

func (e *ETLService) SetEndingBlockHeight(endingBlockHeight int64) {
	e.endingBlockHeight = endingBlockHeight
}

func (e *ETLService) SetRunDownMigrations(runDownMigrations bool) {
	e.runDownMigrations = runDownMigrations
}

func (e *ETLService) SetCheckReadiness(checkReadiness bool) {
	e.checkReadiness = checkReadiness
}

// InitializeChainID fetches and caches the chain ID from the core service
func (e *ETLService) InitializeChainID(ctx context.Context) error {
	nodeInfoResp, err := e.core.GetNodeInfo(ctx, connect.NewRequest(&corev1.GetNodeInfoRequest{}))
	if err != nil {
		// Use fallback chain ID if core service is not available
		e.chainID = "--"
		e.logger.Warn("Failed to get chain ID from core service, using fallback", "error", err, "chainID", e.chainID)
		return nil
	}

	e.chainID = nodeInfoResp.Msg.Chainid
	e.logger.Info("Initialized chain ID", "chainID", e.chainID)
	return nil
}

// GetHealth implements v1connect.ETLServiceHandler.
func (e *ETLService) GetHealth(context.Context, *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error) {
	return connect.NewResponse(&v1.GetHealthResponse{}), nil
}

// GetStats implements v1connect.ETLServiceHandler.
func (e *ETLService) GetStats(ctx context.Context, req *connect.Request[v1.GetStatsRequest]) (*connect.Response[v1.GetStatsResponse], error) {
	stats, ok := e.stats.Get("current")
	if !ok {
		return connect.NewResponse(&v1.GetStatsResponse{
			CurrentBlockHeight:   0,
			ChainId:              e.chainID,
			Bps:                  0,
			Tps:                  0,
			TotalTransactions:    0,
			TransactionBreakdown: []*v1.TransactionTypeBreakdown{},
			SyncStatus: &v1.SyncStatus{
				IsSyncing:           true,
				LatestChainHeight:   0,
				LatestIndexedHeight: 0,
				BlockDelta:          0,
			},
		}), nil
	}

	// Convert cached transaction breakdown to protobuf format
	transactionBreakdown := make([]*v1.TransactionTypeBreakdown, len(stats.TransactionBreakdown))
	for i, row := range stats.TransactionBreakdown {
		transactionBreakdown[i] = &v1.TransactionTypeBreakdown{
			Type:  row.Type,
			Count: row.Count,
		}
	}

	return connect.NewResponse(&v1.GetStatsResponse{
		CurrentBlockHeight:            stats.LatestIndexedHeight,
		ChainId:                       e.chainID,
		Bps:                           stats.BPS,
		Tps:                           stats.TPS,
		TotalTransactions:             stats.TotalTransactions,
		TotalTransactions_24H:         stats.TotalTransactions24h,
		TotalTransactionsPrevious_24H: stats.TotalTransactionsPrevious24h,
		TotalTransactions_7D:          stats.TotalTransactions7d,
		TotalTransactions_30D:         stats.TotalTransactions30d,
		ValidatorCount:                stats.ValidatorCount,
		TransactionBreakdown:          transactionBreakdown,
		SyncStatus: &v1.SyncStatus{
			IsSyncing:           stats.IsSyncing,
			LatestChainHeight:   stats.LatestChainHeight,
			LatestIndexedHeight: stats.LatestIndexedHeight,
			BlockDelta:          stats.LatestChainHeight - stats.LatestIndexedHeight,
		},
	}), nil
}

// Ping implements v1connect.ETLServiceHandler.
func (e *ETLService) Ping(context.Context, *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Message: "pong"}), nil
}

// GetBlocks implements v1connect.ETLServiceHandler.
func (e *ETLService) GetBlocks(ctx context.Context, req *connect.Request[v1.GetBlocksRequest]) (*connect.Response[v1.GetBlocksResponse], error) {
	var blocks []*v1.Block

	// Set default limit if not specified
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 50
	}

	offset := req.Msg.Offset
	if offset < 0 {
		offset = 0
	}

	// If no start/end height specified, return latest blocks with pagination
	if req.Msg.StartHeight == 0 && req.Msg.EndHeight == 0 {
		dbBlocks, err := e.db.GetLatestBlocks(ctx, db.GetLatestBlocksParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}

		// Get total count for has_more calculation
		totalCount, err := e.db.GetTotalBlocksCount(ctx)
		if err != nil {
			return nil, err
		}

		blocks = make([]*v1.Block, len(dbBlocks))
		for i, block := range dbBlocks {
			// Get transactions for this block
			txs, err := e.db.GetBlockTransactions(ctx, block.BlockHeight)
			if err != nil {
				return nil, err
			}

			// Sort by index
			sort.Slice(txs, func(i, j int) bool {
				return txs[i].Index < txs[j].Index
			})

			transactions := make([]*v1.Block_Transaction, len(txs))
			for j, tx := range txs {
				transactions[j] = &v1.Block_Transaction{
					Hash:      tx.TxHash,
					Type:      tx.TxType,
					Index:     uint32(tx.Index),
					Timestamp: timestamppb.New(block.BlockTime.Time),
				}
			}

			blocks[i] = &v1.Block{
				Height:       block.BlockHeight,
				Proposer:     block.ProposerAddress,
				Timestamp:    timestamppb.New(block.BlockTime.Time),
				Transactions: transactions,
			}
		}

		hasMore := int64(offset+limit) < totalCount

		return connect.NewResponse(&v1.GetBlocksResponse{
			Blocks:     blocks,
			HasMore:    hasMore,
			TotalCount: int32(totalCount),
		}), nil
	} else {
		// Handle range-based queries (existing logic would go here)
		// For now, just return empty response
		return connect.NewResponse(&v1.GetBlocksResponse{
			Blocks:     []*v1.Block{},
			HasMore:    false,
			TotalCount: 0,
		}), nil
	}
}

// GetTransactions implements v1connect.ETLServiceHandler.
func (e *ETLService) GetTransactions(ctx context.Context, req *connect.Request[v1.GetTransactionsRequest]) (*connect.Response[v1.GetTransactionsResponse], error) {
	// Set default limit if not specified
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 50
	}

	offset := req.Msg.Offset
	if offset < 0 {
		offset = 0
	}

	dbTxs, err := e.db.GetLatestTransactions(ctx, db.GetLatestTransactionsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	// Get total count for has_more calculation
	totalCount, err := e.db.GetTotalTransactionsCount(ctx)
	if err != nil {
		return nil, err
	}

	transactions := make([]*v1.Block_Transaction, len(dbTxs))
	for i, tx := range dbTxs {
		// Get the block timestamp for this transaction
		block, err := e.db.GetIndexedBlock(ctx, tx.BlockHeight)
		if err != nil {
			return nil, err
		}

		transactions[i] = &v1.Block_Transaction{
			Hash:      tx.TxHash,
			Type:      tx.TxType,
			Index:     uint32(tx.Index),
			Timestamp: timestamppb.New(block.BlockTime.Time),
		}
	}

	hasMore := int64(offset+limit) < totalCount

	return connect.NewResponse(&v1.GetTransactionsResponse{
		Transactions: transactions,
		HasMore:      hasMore,
		TotalCount:   int32(totalCount),
	}), nil
}

// GetTransactionsWithBlockInfo is a helper method for the console to get transactions with block heights
func (e *ETLService) GetTransactionsWithBlockInfo(ctx context.Context) ([]*v1.Block_Transaction, map[string]int64, error) {
	dbTxs, err := e.db.GetLatestTransactions(ctx, db.GetLatestTransactionsParams{
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		return nil, nil, err
	}

	transactions := make([]*v1.Block_Transaction, len(dbTxs))
	blockHeights := make(map[string]int64)

	for i, tx := range dbTxs {
		// Get the block timestamp for this transaction
		block, err := e.db.GetIndexedBlock(ctx, tx.BlockHeight)
		if err != nil {
			return nil, nil, err
		}

		transactions[i] = &v1.Block_Transaction{
			Hash:      tx.TxHash,
			Type:      tx.TxType,
			Index:     uint32(tx.Index),
			Timestamp: timestamppb.New(block.BlockTime.Time),
		}

		blockHeights[tx.TxHash] = tx.BlockHeight
	}

	return transactions, blockHeights, nil
}

// GetTransactionsByAddress implements v1connect.ETLServiceHandler.
func (e *ETLService) GetTransactionsByAddress(ctx context.Context, req *connect.Request[v1.GetTransactionsByAddressRequest]) (*connect.Response[v1.GetTransactionsByAddressResponse], error) {
	// Set default limit if not specified
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 50
	}

	offset := req.Msg.Offset
	if offset < 0 {
		offset = 0
	}

	address := req.Msg.Address
	if address == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("address is required"))
	}

	// Get transactions by address from the database
	dbTxs, err := e.db.GetTransactionsByAddress(ctx, db.GetTransactionsByAddressParams{
		Address: address,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}

	// Convert to API response format
	transactions := make([]*v1.AddressTransaction, len(dbTxs))
	for i, tx := range dbTxs {
		transactions[i] = &v1.AddressTransaction{
			TxHash:       tx.TxHash,
			TxType:       tx.TxType,
			BlockHeight:  tx.BlockHeight,
			Index:        tx.Index,
			Address:      tx.Address,
			RelationType: tx.RelationType,
			BlockTime:    timestamppb.New(tx.BlockTime.Time),
		}
	}

	// For has_more calculation, we need to know if there are more results
	// We'll check if we got the full limit - if so, there might be more
	hasMore := len(dbTxs) == int(limit)

	return connect.NewResponse(&v1.GetTransactionsByAddressResponse{
		Transactions: transactions,
		HasMore:      hasMore,
		TotalCount:   int32(len(dbTxs)), // This is approximate - for exact count we'd need another query
	}), nil
}

func (e *ETLService) GetPlays(ctx context.Context, req *connect.Request[v1.GetPlaysRequest]) (*connect.Response[v1.GetPlaysResponse], error) {
	res := new(v1.GetPlaysResponse)

	switch req.Msg.Query.(type) {
	case *v1.GetPlaysRequest_GetPlaysByAddress:
	case *v1.GetPlaysRequest_GetPlaysByLocation:
	case *v1.GetPlaysRequest_GetPlaysByTimeRange:
	case *v1.GetPlaysRequest_GetPlaysByUser:
	case *v1.GetPlaysRequest_GetPlays:
	}

	return connect.NewResponse(res), nil
}

// GetValidators implements v1connect.ETLServiceHandler.
func (e *ETLService) GetValidators(ctx context.Context, req *connect.Request[v1.GetValidatorsRequest]) (*connect.Response[v1.GetValidatorsResponse], error) {
	// Set default pagination
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := req.Msg.Offset
	if offset < 0 {
		offset = 0
	}

	var validators []*v1.ValidatorInfo
	var totalCount int64

	switch req.Msg.Query.(type) {
	case *v1.GetValidatorsRequest_GetRegisteredValidators:
		// Get currently registered validators (not deregistered)
		dbValidators, err := e.db.GetValidatorRegistrations(ctx)
		if err != nil {
			return nil, err
		}

		// Get all deregistrations to filter out deregistered validators
		deregistrations, err := e.db.GetValidatorDeregistrations(ctx)
		if err != nil {
			return nil, err
		}

		// Create map of deregistered comet addresses for quick lookup
		deregisteredMap := make(map[string]bool)
		for _, dereg := range deregistrations {
			deregisteredMap[dereg.CometAddress] = true
		}

		// Filter active validators and apply pagination
		var activeValidators []db.GetValidatorRegistrationsRow
		for _, validator := range dbValidators {
			if !deregisteredMap[validator.CometAddress] {
				activeValidators = append(activeValidators, validator)
			}
		}

		totalCount = int64(len(activeValidators))

		// Apply pagination
		start := offset
		end := offset + limit
		if start > int32(len(activeValidators)) {
			start = int32(len(activeValidators))
		}
		if end > int32(len(activeValidators)) {
			end = int32(len(activeValidators))
		}

		paginatedValidators := activeValidators[start:end]
		validators = make([]*v1.ValidatorInfo, len(paginatedValidators))

		for i, validator := range paginatedValidators {
			// Get block timestamp for registration time
			block, err := e.db.GetIndexedBlock(ctx, validator.BlockHeight)
			var registeredAt *timestamppb.Timestamp
			if err == nil {
				registeredAt = timestamppb.New(block.BlockTime.Time)
			} else {
				registeredAt = timestamppb.Now() // fallback
			}

			validators[i] = &v1.ValidatorInfo{
				Address:                 validator.Address,
				Endpoint:                validator.Endpoint,
				CometAddress:            validator.CometAddress,
				EthBlock:                validator.EthBlock,
				NodeType:                validator.NodeType,
				Spid:                    validator.Spid,
				CometPubkey:             validator.CometPubkey,
				VotingPower:             validator.VotingPower,
				Status:                  v1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE,
				RegisteredAt:            registeredAt,
				LastActivity:            registeredAt,
				RegistrationBlockHeight: validator.BlockHeight,
				RegistrationTxHash:      validator.TxHash,
			}
		}

	case *v1.GetValidatorsRequest_GetValidatorRegistrations:
		// Get all validator registrations with pagination
		dbValidators, err := e.db.GetValidatorRegistrations(ctx)
		if err != nil {
			return nil, err
		}

		totalCount = int64(len(dbValidators))

		// Apply pagination
		start := offset
		end := offset + limit
		if start > int32(len(dbValidators)) {
			start = int32(len(dbValidators))
		}
		if end > int32(len(dbValidators)) {
			end = int32(len(dbValidators))
		}

		paginatedValidators := dbValidators[start:end]
		validators = make([]*v1.ValidatorInfo, len(paginatedValidators))

		for i, validator := range paginatedValidators {
			// Get block timestamp for registration time
			block, err := e.db.GetIndexedBlock(ctx, validator.BlockHeight)
			var registeredAt *timestamppb.Timestamp
			if err == nil {
				registeredAt = timestamppb.New(block.BlockTime.Time)
			} else {
				registeredAt = timestamppb.Now() // fallback
			}

			validators[i] = &v1.ValidatorInfo{
				Address:                 validator.Address,
				Endpoint:                validator.Endpoint,
				CometAddress:            validator.CometAddress,
				EthBlock:                validator.EthBlock,
				NodeType:                validator.NodeType,
				Spid:                    validator.Spid,
				CometPubkey:             validator.CometPubkey,
				VotingPower:             validator.VotingPower,
				Status:                  v1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE, // Default to active for registrations
				RegisteredAt:            registeredAt,
				LastActivity:            registeredAt,
				RegistrationBlockHeight: validator.BlockHeight,
				RegistrationTxHash:      validator.TxHash,
			}
		}

	case *v1.GetValidatorsRequest_GetValidatorDeregistrations:
		// Get all validator deregistrations with pagination
		dbDeregistrations, err := e.db.GetValidatorDeregistrations(ctx)
		if err != nil {
			return nil, err
		}

		totalCount = int64(len(dbDeregistrations))

		// Apply pagination
		start := offset
		end := offset + limit
		if start > int32(len(dbDeregistrations)) {
			start = int32(len(dbDeregistrations))
		}
		if end > int32(len(dbDeregistrations)) {
			end = int32(len(dbDeregistrations))
		}

		paginatedDeregistrations := dbDeregistrations[start:end]
		validators = make([]*v1.ValidatorInfo, len(paginatedDeregistrations))

		for i, dereg := range paginatedDeregistrations {
			// Get block timestamp for deregistration time
			block, err := e.db.GetIndexedBlock(ctx, dereg.BlockHeight)
			var deregisteredAt *timestamppb.Timestamp
			if err == nil {
				deregisteredAt = timestamppb.New(block.BlockTime.Time)
			} else {
				deregisteredAt = timestamppb.Now() // fallback
			}

			validators[i] = &v1.ValidatorInfo{
				CometAddress:            dereg.CometAddress,
				CometPubkey:             dereg.CometPubkey,
				Status:                  v1.ValidatorStatus_VALIDATOR_STATUS_DEREGISTERED,
				LastActivity:            deregisteredAt,
				RegistrationBlockHeight: dereg.BlockHeight,
				RegistrationTxHash:      dereg.TxHash,
			}
		}

	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	hasMore := int64(offset+limit) < totalCount

	return connect.NewResponse(&v1.GetValidatorsResponse{
		Validators: validators,
		HasMore:    hasMore,
		TotalCount: int32(totalCount),
	}), nil
}

// GetManageEntities implements v1connect.ETLServiceHandler.
func (e *ETLService) GetManageEntities(context.Context, *connect.Request[v1.GetManageEntitiesRequest]) (*connect.Response[v1.GetManageEntitiesResponse], error) {
	res := new(v1.GetManageEntitiesResponse)
	return connect.NewResponse(res), nil
}

// GetLocation implements v1connect.ETLServiceHandler.
func (e *ETLService) GetLocation(context.Context, *connect.Request[v1.GetLocationRequest]) (*connect.Response[v1.GetLocationResponse], error) {
	res := new(v1.GetLocationResponse)
	return connect.NewResponse(res), nil
}

// GetBlock implements v1connect.ETLServiceHandler.
func (e *ETLService) GetBlock(ctx context.Context, req *connect.Request[v1.GetBlockRequest]) (*connect.Response[v1.GetBlockResponse], error) {
	if req.Msg.Height <= 0 {
		height, err := e.db.GetLatestIndexedBlock(ctx)
		if err != nil {
			return nil, err
		}
		req.Msg.Height = height
	}

	block, err := e.db.GetIndexedBlock(ctx, req.Msg.Height)
	if err != nil {
		return nil, err
	}

	txs, err := e.db.GetBlockTransactions(ctx, req.Msg.Height)
	if err != nil {
		return nil, err
	}

	// sort by index
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Index < txs[j].Index
	})

	transactions := make([]*v1.Block_Transaction, len(txs))
	for i, tx := range txs {
		transactions[i] = &v1.Block_Transaction{
			Hash:      tx.TxHash,
			Type:      tx.TxType,
			Index:     uint32(tx.Index),
			Timestamp: timestamppb.New(block.BlockTime.Time),
		}
	}

	return connect.NewResponse(&v1.GetBlockResponse{
		Block: &v1.Block{
			Height:       block.BlockHeight,
			Proposer:     block.ProposerAddress,
			Timestamp:    timestamppb.New(block.BlockTime.Time),
			Transactions: transactions,
		},
	}), nil
}

func NewETLService(core corev1connect.CoreServiceClient, logger *common.Logger) *ETLService {
	etl := &ETLService{
		logger:      logger.Child("etl"),
		core:        core,
		blockPubsub: NewPubsub[*v1.Block](),
		playPubsub:  NewPubsub[*v1.TrackPlay](),
	}

	return etl
}

// GetTransaction implements v1connect.ETLServiceHandler.
func (e *ETLService) GetTransaction(ctx context.Context, req *connect.Request[v1.GetTransactionRequest]) (*connect.Response[v1.GetTransactionResponse], error) {
	txHash := req.Msg.TxHash
	if txHash == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	// Get the transaction directly by hash
	txResult, err := e.db.GetTransaction(ctx, txHash)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}

	// Create the base transaction
	transaction := &v1.Transaction{
		Hash:            txResult.TxHash,
		Type:            txResult.TxType,
		BlockHeight:     txResult.BlockHeight,
		Index:           txResult.Index,
		BlockTime:       timestamppb.New(txResult.BlockTime.Time),
		ProposerAddress: txResult.ProposerAddress,
	}

	// Get transaction content based on type using direct queries by tx_hash
	switch txResult.TxType {
	case TxTypePlay:
		plays, err := e.db.GetPlaysByTxHash(ctx, txHash)
		if err != nil {
			return nil, err
		}

		trackPlays := make([]*v1.TrackPlay, len(plays))
		for i, play := range plays {
			trackPlays[i] = &v1.TrackPlay{
				Address:  play.Address,
				TrackId:  play.TrackID,
				PlayedAt: timestamppb.New(time.Unix(play.Timestamp, 0)),
				City:     play.City,
				Region:   play.Region,
				Country:  play.Country,
			}
		}
		transaction.Content = &v1.Transaction_Plays{
			Plays: &v1.TrackPlaysTransaction{Plays: trackPlays},
		}

	case TxTypeManageEntity:
		entities, err := e.db.GetManageEntitiesByTxHash(ctx, txHash)
		if err != nil {
			return nil, err
		}

		manageEntities := make([]*v1.ManageEntity, len(entities))
		for i, entity := range entities {
			manageEntities[i] = &v1.ManageEntity{
				Address:    entity.Address,
				EntityType: entity.EntityType,
				EntityId:   entity.EntityID,
				Action:     entity.Action,
				Metadata:   entity.Metadata.String,
				Signature:  entity.Signature,
				Signer:     entity.Signer,
				Nonce:      entity.Nonce,
			}
		}
		transaction.Content = &v1.Transaction_ManageEntity{
			ManageEntity: &v1.ManageEntityTransaction{Entities: manageEntities},
		}

	case TxTypeValidatorRegistration:
		registrations, err := e.db.GetValidatorRegistrationsByTxHash(ctx, txHash)
		if err != nil {
			return nil, err
		}

		validatorRegistrations := make([]*v1.ValidatorRegistration, len(registrations))
		for i, reg := range registrations {
			validatorRegistrations[i] = &v1.ValidatorRegistration{
				Address:      reg.Address,
				CometAddress: reg.CometAddress,
				EthBlock:     reg.EthBlock,
				NodeType:     reg.NodeType,
				Spid:         reg.Spid,
				CometPubkey:  reg.CometPubkey,
				VotingPower:  reg.VotingPower,
			}
		}
		transaction.Content = &v1.Transaction_ValidatorRegistration{
			ValidatorRegistration: &v1.ValidatorRegistrationTransaction{Registrations: validatorRegistrations},
		}

	case TxTypeValidatorDeregistration:
		deregistrations, err := e.db.GetValidatorDeregistrationsByTxHash(ctx, txHash)
		if err != nil {
			return nil, err
		}

		validatorDeregistrations := make([]*v1.ValidatorDeregistration, len(deregistrations))
		for i, dereg := range deregistrations {
			validatorDeregistrations[i] = &v1.ValidatorDeregistration{
				CometAddress: dereg.CometAddress,
				CometPubkey:  dereg.CometPubkey,
			}
		}
		transaction.Content = &v1.Transaction_ValidatorDeregistration{
			ValidatorDeregistration: &v1.ValidatorDeregistrationTransaction{Deregistrations: validatorDeregistrations},
		}

	case TxTypeValidatorRegistrationLegacy:
		// Legacy validator registration uses the same structure as regular validator registration
		registrations, err := e.db.GetValidatorRegistrationsByTxHash(ctx, txHash)
		if err != nil {
			return nil, err
		}

		validatorRegistrations := make([]*v1.ValidatorRegistration, len(registrations))
		for i, reg := range registrations {
			validatorRegistrations[i] = &v1.ValidatorRegistration{
				Address:      reg.Address,
				CometAddress: reg.CometAddress,
				EthBlock:     reg.EthBlock,
				NodeType:     reg.NodeType,
				Spid:         reg.Spid,
				CometPubkey:  reg.CometPubkey,
				VotingPower:  reg.VotingPower,
			}
		}
		transaction.Content = &v1.Transaction_ValidatorRegistration{
			ValidatorRegistration: &v1.ValidatorRegistrationTransaction{Registrations: validatorRegistrations},
		}

	case TxTypeSlaRollup:
		slaRollups, err := e.db.GetSlaRollupsByTxHash(ctx, txHash)
		if err != nil {
			return nil, err
		}

		slaNodeReports, err := e.db.GetSlaNodeReportsByTxHash(ctx, txHash)
		if err != nil {
			return nil, err
		}

		// Convert to protobuf structures
		reports := make([]*v1.SlaNodeReport, len(slaNodeReports))
		for i, report := range slaNodeReports {
			reports[i] = &v1.SlaNodeReport{
				Address:           report.Address,
				NumBlocksProposed: report.NumBlocksProposed,
			}
		}

		// For SLA rollup transaction, we expect one rollup record
		var timestamp *timestamppb.Timestamp
		var blockStart, blockEnd int64
		if len(slaRollups) > 0 {
			rollup := slaRollups[0]
			timestamp = timestamppb.New(rollup.Timestamp.Time)
			blockStart = rollup.BlockStart
			blockEnd = rollup.BlockEnd
		}

		transaction.Content = &v1.Transaction_SlaRollup{
			SlaRollup: &v1.SlaRollupTransaction{
				Timestamp:  timestamp,
				BlockStart: blockStart,
				BlockEnd:   blockEnd,
				Reports:    reports,
			},
		}

	case TxTypeStorageProof:
		storageProofs, err := e.db.GetStorageProofsByTxHash(ctx, txHash)
		if err != nil {
			e.logger.Error("Failed to get storage proofs", "error", err, "txHash", txHash)
			return nil, err
		}

		e.logger.Info("Storage proof query result", "txHash", txHash, "count", len(storageProofs))

		// For storage proof transaction, we expect one proof record
		if len(storageProofs) > 0 {
			proof := storageProofs[0]
			e.logger.Info("Storage proof data", "height", proof.Height, "address", proof.Address, "cid", proof.Cid)
			transaction.Content = &v1.Transaction_StorageProof{
				StorageProof: &v1.StorageProofTransaction{
					Height:          proof.Height,
					Address:         proof.Address,
					ProverAddresses: proof.ProverAddresses,
					Cid:             proof.Cid,
					ProofSignature:  proof.ProofSignature,
				},
			}
		} else {
			e.logger.Warn("No storage proofs found for transaction", "txHash", txHash)
			// Empty storage proof if no records found
			transaction.Content = &v1.Transaction_StorageProof{
				StorageProof: &v1.StorageProofTransaction{},
			}
		}

	case TxTypeStorageProofVerification:
		verifications, err := e.db.GetStorageProofVerificationsByTxHash(ctx, txHash)
		if err != nil {
			e.logger.Error("Failed to get storage proof verifications", "error", err, "txHash", txHash)
			return nil, err
		}

		e.logger.Info("Storage proof verification query result", "txHash", txHash, "count", len(verifications))

		// For storage proof verification transaction, we expect one verification record
		if len(verifications) > 0 {
			verification := verifications[0]
			e.logger.Info("Storage proof verification data", "height", verification.Height, "proofLength", len(verification.Proof))
			transaction.Content = &v1.Transaction_StorageProofVerification{
				StorageProofVerification: &v1.StorageProofVerificationTransaction{
					Height: verification.Height,
					Proof:  verification.Proof,
				},
			}
		} else {
			e.logger.Warn("No storage proof verifications found for transaction", "txHash", txHash)
			// Empty storage proof verification if no records found
			transaction.Content = &v1.Transaction_StorageProofVerification{
				StorageProofVerification: &v1.StorageProofVerificationTransaction{},
			}
		}

	case TxTypeRelease:
		releases, err := e.db.GetReleasesByTxHash(ctx, txHash)
		if err != nil {
			return nil, err
		}

		// For release transaction, we expect one release record
		if len(releases) > 0 {
			release := releases[0]
			transaction.Content = &v1.Transaction_Release{
				Release: &v1.ReleaseTransaction{
					ReleaseData: release.ReleaseData,
				},
			}
		} else {
			// Empty release if no records found
			transaction.Content = &v1.Transaction_Release{
				Release: &v1.ReleaseTransaction{},
			}
		}

	case TxTypeValidatorMisbehaviorDeregistration:
		// This likely uses the same structure as regular validator deregistration
		deregistrations, err := e.db.GetValidatorDeregistrationsByTxHash(ctx, txHash)
		if err != nil {
			return nil, err
		}

		validatorDeregistrations := make([]*v1.ValidatorDeregistration, len(deregistrations))
		for i, dereg := range deregistrations {
			validatorDeregistrations[i] = &v1.ValidatorDeregistration{
				CometAddress: dereg.CometAddress,
				CometPubkey:  dereg.CometPubkey,
			}
		}
		transaction.Content = &v1.Transaction_ValidatorDeregistration{
			ValidatorDeregistration: &v1.ValidatorDeregistrationTransaction{Deregistrations: validatorDeregistrations},
		}

	default:
		// For unknown transaction types, don't set content
	}

	return connect.NewResponse(&v1.GetTransactionResponse{
		Transaction: transaction,
	}), nil
}

// Search implements v1connect.ETLServiceHandler.
func (e *ETLService) Search(ctx context.Context, req *connect.Request[v1.SearchRequest]) (*connect.Response[v1.SearchResponse], error) {
	query := req.Msg.Query
	if query == "" {
		return connect.NewResponse(&v1.SearchResponse{Results: []*v1.SearchResult{}}), nil
	}

	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 20
	}

	// Use the unified search query for better results
	searchResults, err := e.db.SearchUnified(ctx, db.SearchUnifiedParams{
		Column1: query,
		Limit:   limit,
	})
	if err != nil {
		e.logger.Error("Error in unified search", "error", err, "query", query)
		// Fallback to individual searches if unified search fails
		return e.fallbackSearch(ctx, query, limit)
	}

	// Convert to protobuf response
	results := make([]*v1.SearchResult, len(searchResults))
	for i, result := range searchResults {
		var url string
		switch result.Type {
		case "block":
			url = "/block/" + result.ID
		case "transaction":
			url = "/transaction/" + result.ID
		case "account":
			url = "/account/" + result.ID
		case "validator":
			url = "/validator/" + result.ID
		default:
			url = ""
		}

		// Safe type assertions with fallbacks
		title, titleOk := result.Title.(string)
		if !titleOk {
			title = fmt.Sprintf("%v", result.Title)
		}

		subtitle, subtitleOk := result.Subtitle.(string)
		if !subtitleOk {
			subtitle = fmt.Sprintf("%v", result.Subtitle)
		}

		results[i] = &v1.SearchResult{
			Id:       result.ID,
			Title:    title,
			Subtitle: subtitle,
			Type:     result.Type,
			Url:      url,
		}
	}

	return connect.NewResponse(&v1.SearchResponse{Results: results}), nil
}

// fallbackSearch provides individual searches if unified search fails
func (e *ETLService) fallbackSearch(ctx context.Context, query string, limit int32) (*connect.Response[v1.SearchResponse], error) {
	var results []*v1.SearchResult

	// Search blocks if query looks like a number
	if blockHeight, err := strconv.ParseInt(query, 10, 64); err == nil && blockHeight > 0 {
		e.logger.Info("Searching for block height", "blockHeight", blockHeight, "query", query)
		blockHeights, err := e.db.SearchBlockHeight(ctx, query) // Now passing string
		if err == nil {
			e.logger.Info("Block search results", "count", len(blockHeights), "results", blockHeights)
			for i, bh := range blockHeights {
				if i >= 5 { // Limit to 5 results per type
					break
				}
				results = append(results, &v1.SearchResult{
					Id:       strconv.FormatInt(bh, 10),
					Title:    fmt.Sprintf("Block #%d", bh),
					Subtitle: "Block",
					Type:     "block",
					Url:      fmt.Sprintf("/block/%d", bh),
				})
			}
		} else {
			e.logger.Error("Error searching block heights", "error", err, "query", query)
		}
	}

	// Search transactions if query looks like a hash
	if strings.HasPrefix(query, "0x") && len(query) > 10 {
		txHashes, err := e.db.SearchTxHash(ctx, query)
		if err == nil {
			for i, hash := range txHashes {
				if i >= 5 { // Limit to 5 results per type
					break
				}
				results = append(results, &v1.SearchResult{
					Id:       hash,
					Title:    hash[:20] + "...",
					Subtitle: "Transaction",
					Type:     "transaction",
					Url:      "/transaction/" + hash,
				})
			}
		} else {
			e.logger.Error("Error searching transactions", "error", err, "query", query)
		}
	}

	// Search addresses
	if strings.HasPrefix(query, "0x") || len(query) >= 8 {
		addresses, err := e.db.SearchAddress(ctx, query)
		if err == nil {
			for i, addr := range addresses {
				if i >= 5 { // Limit to 5 results per type
					break
				}
				results = append(results, &v1.SearchResult{
					Id:       addr,
					Title:    addr[:20] + "...",
					Subtitle: "Account address",
					Type:     "account",
					Url:      "/account/" + addr,
				})
			}
		} else {
			e.logger.Error("Error searching addresses", "error", err, "query", query)
		}
	}

	// Search validators
	if len(query) >= 8 {
		validatorAddresses, err := e.db.SearchValidatorRegistration(ctx, query)
		if err == nil {
			for i, addr := range validatorAddresses {
				if i >= 5 { // Limit to 5 results per type
					break
				}
				results = append(results, &v1.SearchResult{
					Id:       addr,
					Title:    addr[:20] + "...",
					Subtitle: "Validator",
					Type:     "validator",
					Url:      "/validator/" + addr,
				})
			}
		} else {
			e.logger.Error("Error searching validators", "error", err, "query", query)
		}
	}

	// Limit total results
	if len(results) > int(limit) {
		results = results[:limit]
	}

	return connect.NewResponse(&v1.SearchResponse{Results: results}), nil
}

// GetTransactionsForAPI is a method for API endpoints that includes block heights
func (e *ETLService) GetTransactionsForAPI(ctx context.Context, limit, offset int32) (*v1.GetTransactionsResponse, map[string]int64, error) {
	response, err := e.GetTransactions(ctx, &connect.Request[v1.GetTransactionsRequest]{
		Msg: &v1.GetTransactionsRequest{
			Limit:  limit,
			Offset: offset,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	// Create block heights map
	blockHeights := make(map[string]int64)
	dbTxs, err := e.db.GetLatestTransactions(ctx, db.GetLatestTransactionsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return response.Msg, blockHeights, nil // Return what we have if getting block heights fails
	}

	for _, tx := range dbTxs {
		blockHeights[tx.TxHash] = tx.BlockHeight
	}

	return response.Msg, blockHeights, nil
}

// GetValidator implements v1connect.ETLServiceHandler.
func (e *ETLService) GetValidator(ctx context.Context, req *connect.Request[v1.GetValidatorRequest]) (*connect.Response[v1.GetValidatorResponse], error) {
	var validatorInfo *v1.ValidatorInfo
	var events []*v1.ValidatorEvent

	// Get all validator registrations, deregistrations to find the requested validator
	registrations, err := e.db.GetValidatorRegistrations(ctx)
	if err != nil {
		return nil, err
	}

	deregistrations, err := e.db.GetValidatorDeregistrations(ctx)
	if err != nil {
		return nil, err
	}

	// Find the validator based on the identifier
	var targetRegistration *db.GetValidatorRegistrationsRow
	var targetCometAddress string

	switch req.Msg.Identifier.(type) {
	case *v1.GetValidatorRequest_Address:
		address := req.Msg.GetAddress()
		for _, reg := range registrations {
			if reg.Address == address {
				targetRegistration = &reg
				targetCometAddress = reg.CometAddress
				break
			}
		}
	case *v1.GetValidatorRequest_CometAddress:
		cometAddress := req.Msg.GetCometAddress()
		targetCometAddress = cometAddress
		for _, reg := range registrations {
			if reg.CometAddress == cometAddress {
				targetRegistration = &reg
				break
			}
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	if targetRegistration == nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}

	// Get registration block timestamp
	registrationBlock, err := e.db.GetIndexedBlock(ctx, targetRegistration.BlockHeight)
	var registeredAt *timestamppb.Timestamp
	if err == nil {
		registeredAt = timestamppb.New(registrationBlock.BlockTime.Time)
	} else {
		registeredAt = timestamppb.Now() // fallback
	}

	// Determine current status by checking if deregistered
	status := v1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE
	var lastActivity *timestamppb.Timestamp = registeredAt

	for _, dereg := range deregistrations {
		if dereg.CometAddress == targetCometAddress {
			status = v1.ValidatorStatus_VALIDATOR_STATUS_DEREGISTERED
			// Get deregistration block timestamp
			deregBlock, err := e.db.GetIndexedBlock(ctx, dereg.BlockHeight)
			if err == nil {
				lastActivity = timestamppb.New(deregBlock.BlockTime.Time)
			}
			break
		}
	}

	// Build validator info
	validatorInfo = &v1.ValidatorInfo{
		Address:                 targetRegistration.Address,
		Endpoint:                targetRegistration.Endpoint,
		CometAddress:            targetRegistration.CometAddress,
		EthBlock:                targetRegistration.EthBlock,
		NodeType:                targetRegistration.NodeType,
		Spid:                    targetRegistration.Spid,
		CometPubkey:             targetRegistration.CometPubkey,
		VotingPower:             targetRegistration.VotingPower,
		Status:                  status,
		RegisteredAt:            registeredAt,
		LastActivity:            lastActivity,
		RegistrationBlockHeight: targetRegistration.BlockHeight,
		RegistrationTxHash:      targetRegistration.TxHash,
	}

	// Build event history
	events = []*v1.ValidatorEvent{}

	// Add registration event
	events = append(events, &v1.ValidatorEvent{
		Type:        v1.ValidatorEventType_VALIDATOR_EVENT_TYPE_REGISTRATION,
		Timestamp:   registeredAt,
		BlockHeight: targetRegistration.BlockHeight,
		TxHash:      targetRegistration.TxHash,
		Data: &v1.ValidatorEventData{
			Event: &v1.ValidatorEventData_Registration{
				Registration: &v1.ValidatorRegistrationEvent{
					Address:      targetRegistration.Address,
					Endpoint:     "", // Not available in query result
					CometAddress: targetRegistration.CometAddress,
					EthBlock:     targetRegistration.EthBlock,
					NodeType:     targetRegistration.NodeType,
					Spid:         targetRegistration.Spid,
					CometPubkey:  targetRegistration.CometPubkey,
					VotingPower:  targetRegistration.VotingPower,
				},
			},
		},
	})

	// Add deregistration event if applicable
	for _, dereg := range deregistrations {
		if dereg.CometAddress == targetCometAddress {
			deregBlock, err := e.db.GetIndexedBlock(ctx, dereg.BlockHeight)
			var deregTime *timestamppb.Timestamp
			if err == nil {
				deregTime = timestamppb.New(deregBlock.BlockTime.Time)
			} else {
				deregTime = timestamppb.Now() // fallback
			}

			events = append(events, &v1.ValidatorEvent{
				Type:        v1.ValidatorEventType_VALIDATOR_EVENT_TYPE_DEREGISTRATION,
				Timestamp:   deregTime,
				BlockHeight: dereg.BlockHeight,
				TxHash:      dereg.TxHash,
				Data: &v1.ValidatorEventData{
					Event: &v1.ValidatorEventData_Deregistration{
						Deregistration: &v1.ValidatorDeregistrationEvent{
							CometAddress: dereg.CometAddress,
							CometPubkey:  dereg.CometPubkey,
						},
					},
				},
			})
			break
		}
	}

	return connect.NewResponse(&v1.GetValidatorResponse{
		Validator: validatorInfo,
		Events:    events,
	}), nil
}

// Stream implements v1connect.ETLServiceHandler.
func (e *ETLService) Stream(ctx context.Context, stream *connect.BidiStream[v1.StreamRequest, v1.StreamResponse]) error {
	var blockCh chan *v1.Block
	var playCh chan *v1.TrackPlay

	// Handle incoming stream requests
	go func() {
		for {
			req, err := stream.Receive()
			if err != nil {
				// Client closed connection or other error
				return
			}

			switch req.Query.(type) {
			case *v1.StreamRequest_StreamBlocks:
				// Subscribe to blocks if not already subscribed
				if blockCh == nil {
					blockCh = e.blockPubsub.Subscribe(BlockTopic, 100)
					e.logger.Info("Subscribed to block stream")
				}
			case *v1.StreamRequest_StreamPlays:
				// Subscribe to plays if not already subscribed
				if playCh == nil {
					playCh = e.playPubsub.Subscribe(PlayTopic, 100)
					e.logger.Info("Subscribed to play stream")
				}
			}
		}
	}()

	// Handle outgoing messages from pubsub
	for {
		select {
		case <-ctx.Done():
			// Cleanup subscriptions when context is cancelled
			if blockCh != nil {
				e.blockPubsub.Unsubscribe(BlockTopic, blockCh)
				e.logger.Info("Unsubscribed from block stream")
			}
			if playCh != nil {
				e.playPubsub.Unsubscribe(PlayTopic, playCh)
				e.logger.Info("Unsubscribed from play stream")
			}
			return ctx.Err()

		case block := <-blockCh:
			if block != nil {
				// Send block data as StreamBlocksResponse
				err := stream.Send(&v1.StreamResponse{
					Response: &v1.StreamResponse_StreamBlocks{
						StreamBlocks: &v1.StreamResponse_StreamBlocksResponse{
							Height:   block.Height,
							Proposer: block.Proposer,
						},
					},
				})
				if err != nil {
					e.logger.Error("Failed to send block stream response", "error", err)
					return err
				}
			}

		case play := <-playCh:
			if play != nil {
				// Send play data as StreamPlaysResponse
				err := stream.Send(&v1.StreamResponse{
					Response: &v1.StreamResponse_StreamPlays{
						StreamPlays: &v1.StreamResponse_StreamPlaysResponse{
							City:      play.City,
							Country:   play.Country,
							Region:    play.Region,
							Latitude:  play.Latitude,
							Longitude: play.Longitude,
						},
					},
				})
				if err != nil {
					e.logger.Error("Failed to send play stream response", "error", err)
					return err
				}
			}
		}
	}
}

// GetPlayPubsub returns the play pubsub for external subscribers
func (e *ETLService) GetPlayPubsub() *PlayPubsub {
	return e.playPubsub
}

// GetBlockPubsub returns the block pubsub for external subscribers
func (e *ETLService) GetBlockPubsub() *BlockPubsub {
	return e.blockPubsub
}
