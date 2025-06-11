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
func (e *ETLService) GetBlocks(context.Context, *connect.Request[v1.GetBlocksRequest]) (*connect.Response[v1.GetBlocksResponse], error) {
	res := new(v1.GetBlocksResponse)
	return connect.NewResponse(res), nil
}

// GetTransactions implements v1connect.ETLServiceHandler.
func (e *ETLService) GetTransactions(context.Context, *connect.Request[v1.GetTransactionsRequest]) (*connect.Response[v1.GetTransactionsResponse], error) {
	res := new(v1.GetTransactionsResponse)
	return connect.NewResponse(res), nil
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
