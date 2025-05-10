package etl

import (
	"context"

	"connectrpc.com/connect"
	corev1connect "github.com/AudiusProject/audiusd/pkg/api/core/v1/v1connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/etl/v1"
	"github.com/AudiusProject/audiusd/pkg/api/etl/v1/v1connect"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/etl/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ v1connect.ETLServiceHandler = (*ETLService)(nil)

type ETLService struct {
	dbURL  string
	core   corev1connect.CoreServiceClient
	pool   *pgxpool.Pool
	db     *db.Queries
	logger *common.Logger
}

func NewETLService(core corev1connect.CoreServiceClient, logger *common.Logger) *ETLService {
	return &ETLService{
		logger: logger.Child("etl"),
		core:   core,
	}
}

func (e *ETLService) SetDBURL(dbURL string) {
	e.dbURL = dbURL
}

// GetHealth implements v1connect.ETLServiceHandler.
func (e *ETLService) GetHealth(context.Context, *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error) {
	return connect.NewResponse(&v1.GetHealthResponse{}), nil
}

// Ping implements v1connect.ETLServiceHandler.
func (e *ETLService) Ping(context.Context, *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Message: "pong"}), nil
}

func (e *ETLService) GetPlays(ctx context.Context, req *connect.Request[v1.GetPlaysRequest]) (*connect.Response[v1.GetPlaysResponse], error) {
	return connect.NewResponse(&v1.GetPlaysResponse{}), nil
}
