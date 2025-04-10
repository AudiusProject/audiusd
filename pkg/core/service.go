package core

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/api/core/v1/v1connect"
	"github.com/AudiusProject/audiusd/pkg/core/db"
)

// CoreService implements the v1connect.CoreService interface
var _ v1connect.CoreServiceHandler = (*CoreService)(nil)

type CoreService struct {
	db *db.Queries
}

func (c *CoreService) GetBlock(ctx context.Context, req *connect.Request[v1.GetBlockRequest]) (*connect.Response[v1.GetBlockResponse], error) {
	blockNumber := req.Msg.Height
	return connect.NewResponse(&v1.GetBlockResponse{Block: &v1.Block{Height: blockNumber}}), nil
}

func NewCoreService(db *db.Queries) *CoreService {
	return &CoreService{
		db: db,
	}
}
