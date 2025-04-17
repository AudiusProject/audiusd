package system

import (
	"context"

	"connectrpc.com/connect"
	coreV1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	etlV1 "github.com/AudiusProject/audiusd/pkg/api/etl/v1"
	storageV1 "github.com/AudiusProject/audiusd/pkg/api/storage/v1"
	v1 "github.com/AudiusProject/audiusd/pkg/api/system/v1"
	"github.com/AudiusProject/audiusd/pkg/api/system/v1/v1connect"
	"github.com/AudiusProject/audiusd/pkg/core/server"
	"github.com/AudiusProject/audiusd/pkg/etl"
	storageServer "github.com/AudiusProject/audiusd/pkg/mediorum/server"
	"golang.org/x/sync/errgroup"
)

type SystemService struct {
	core    *server.CoreService
	storage *storageServer.StorageService
	etl     *etl.ETLService
}

var _ v1connect.SystemServiceHandler = (*SystemService)(nil)

func NewSystemService(core *server.CoreService, storage *storageServer.StorageService, etl *etl.ETLService) *SystemService {
	return &SystemService{core: core, storage: storage, etl: etl}
}

// GetHealth implements v1connect.SystemServiceHandler.
func (s *SystemService) GetHealth(context.Context, *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error) {
	return connect.NewResponse(&v1.GetHealthResponse{}), nil
}

// Ping implements v1connect.SystemServiceHandler.
func (s *SystemService) Ping(ctx context.Context, _req *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error) {
	res := &v1.PingResponse{Message: "pong"}

	g := errgroup.Group{}

	g.Go(func() error {
		corePing, err := s.core.Ping(ctx, connect.NewRequest(&coreV1.PingRequest{}))
		if err != nil {
			return err
		}
		res.CorePing = corePing.Msg
		return nil
	})

	g.Go(func() error {
		storagePing, err := s.storage.Ping(ctx, connect.NewRequest(&storageV1.PingRequest{}))
		if err != nil {
			return err
		}
		res.StoragePing = storagePing.Msg
		return nil
	})

	g.Go(func() error {
		etlPing, err := s.etl.Ping(ctx, connect.NewRequest(&etlV1.PingRequest{}))
		if err != nil {
			return err
		}
		res.EtlPing = etlPing.Msg
		return nil
	})

	err := g.Wait()
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(res), nil
}
