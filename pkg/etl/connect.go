package etl

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	corev1connect "github.com/AudiusProject/audiusd/pkg/api/core/v1/v1connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/etl/v1"
	"github.com/AudiusProject/audiusd/pkg/api/etl/v1/v1connect"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/etl/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ v1connect.ETLServiceHandler = (*ETLService)(nil)

type ETLService struct {
	dbURL               string
	runDownMigrations   bool
	startingBlockHeight int64
	endingBlockHeight   int64
	checkReadiness      bool

	core   corev1connect.CoreServiceClient
	pool   *pgxpool.Pool
	db     *db.Queries
	logger *common.Logger
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

// GetHealth implements v1connect.ETLServiceHandler.
func (e *ETLService) GetHealth(context.Context, *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error) {
	return connect.NewResponse(&v1.GetHealthResponse{}), nil
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
				Endpoint:                "", // Not available in query result
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
				Endpoint:                "", // Not available in query result
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

// StreamBlocks implements v1connect.ETLServiceHandler.
func (e *ETLService) StreamBlocks(context.Context, *connect.Request[v1.StreamBlocksRequest], *connect.ServerStream[v1.StreamBlocksResponse]) error {
	panic("unimplemented")
}

func NewETLService(core corev1connect.CoreServiceClient, logger *common.Logger) *ETLService {
	return &ETLService{
		logger: logger.Child("etl"),
		core:   core,
	}
}

// GetTransaction implements v1connect.ETLServiceHandler.
func (e *ETLService) GetTransaction(context.Context, *connect.Request[v1.GetTransactionRequest]) (*connect.Response[v1.GetTransactionResponse], error) {
	panic("unimplemented")
}

// StreamTransactions implements v1connect.ETLServiceHandler.
func (e *ETLService) StreamTransactions(context.Context, *connect.Request[v1.StreamTransactionsRequest], *connect.ServerStream[v1.StreamTransactionsResponse]) error {
	panic("unimplemented")
}

// Search implements v1connect.ETLServiceHandler.
func (e *ETLService) Search(context.Context, *connect.Request[v1.SearchRequest]) (*connect.Response[v1.SearchResponse], error) {
	panic("unimplemented")
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
		Endpoint:                "", // Not available in query result
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
