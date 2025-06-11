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
	res := new(v1.GetValidatorsResponse)

	switch req.Msg.Query.(type) {
	case *v1.GetValidatorsRequest_GetRegisteredValidators:
	case *v1.GetValidatorsRequest_GetValidatorDeregistrations:
	case *v1.GetValidatorsRequest_GetValidatorRegistrations:
	}

	return connect.NewResponse(res), nil
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
