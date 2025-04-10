package etl

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/etl/v1"
	"github.com/AudiusProject/audiusd/pkg/api/etl/v1/v1connect"
	"github.com/AudiusProject/audiusd/pkg/core/db"
)

var _ v1connect.ETLServiceHandler = (*ETLService)(nil)

type ETLService struct {
	db *db.Queries
}

func NewETLService(db *db.Queries) *ETLService {
	return &ETLService{db: db}
}

// GetPlays implements v1connect.ETLServiceHandler.
func (e *ETLService) GetPlays(context.Context, *connect.Request[v1.GetPlaysRequest]) (*connect.Response[v1.GetPlaysResponse], error) {
	return &connect.Response[v1.GetPlaysResponse]{
		Msg: &v1.GetPlaysResponse{
			Plays: []*v1.GetPlayResponse{{
				UserId:      "1",
				TrackId:     "2",
				Timestamp:   1234567890,
				City:        "New York",
				Country:     "United States",
				Region:      "NY",
				BlockHeight: 1234567890,
				TxHash:      "0x1234567890",
			}},
		},
	}, nil
}
