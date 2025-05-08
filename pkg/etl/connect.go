package etl

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"
	corev1connect "github.com/AudiusProject/audiusd/pkg/api/core/v1/v1connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/etl/v1"
	"github.com/AudiusProject/audiusd/pkg/api/etl/v1/v1connect"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ v1connect.ETLServiceHandler = (*ETLService)(nil)

type ETLService struct {
	core corev1connect.CoreServiceClient
	db   ETLDatabase
}

func NewETLService(core corev1connect.CoreServiceClient, db ETLDatabase) *ETLService {
	return &ETLService{
		db: db,
	}
}

func DefaultETLService() (*ETLService, error) {
	dbUrl := os.Getenv("dbUrl")
	if dbUrl == "" {
		return nil, fmt.Errorf("dbUrl environment variable not set")
	}

	pgConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, fmt.Errorf("error parsing database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), pgConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating database pool: %v", err)
	}

	return NewETLService(NewPostgresETLWriter(pool)), nil
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
	plays, err := e.db.GetPlays(ctx, req.Msg.UserId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.GetPlaysResponse{Plays: plays}), nil
}
