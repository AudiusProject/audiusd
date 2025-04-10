package mediorum

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/storage/v1"
	"github.com/AudiusProject/audiusd/pkg/api/storage/v1/v1connect"
)

// StorageService is a service that handles storage operations.
var _ v1connect.StorageServiceHandler = (*StorageService)(nil)

type StorageService struct {
}

func (s *StorageService) GetSignedStreamUrl(ctx context.Context, req *connect.Request[v1.GetSignedStreamUrlRequest]) (*connect.Response[v1.GetSignedStreamUrlResponse], error) {
	// Mock JWT verification
	if req.Msg.Jwt != "Bearer valid-jwt-token" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid JWT"))
	}

	// Mock CID lookup
	cid := req.Msg.Cid
	if cid == "" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("CID not found"))
	}

	// Return mock AWS presigned URL
	mockUrl := "https://mock.aws.s3/presigned-url-for-" + cid
	response := &v1.GetSignedStreamUrlResponse{
		Url: mockUrl,
	}

	return connect.NewResponse(response), nil
}
