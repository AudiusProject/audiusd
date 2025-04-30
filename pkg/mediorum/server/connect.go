package server

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/storage/v1"
	"github.com/AudiusProject/audiusd/pkg/api/storage/v1/v1connect"
)

var _ v1connect.StorageServiceHandler = (*StorageService)(nil)

type StorageService struct {
	mediorum *MediorumServer
}

func NewStorageService() *StorageService {
	return &StorageService{}
}

func (s *StorageService) SetMediorum(mediorum *MediorumServer) {
	s.mediorum = mediorum
}

// GetHealth implements v1connect.StorageServiceHandler.
func (s *StorageService) GetHealth(context.Context, *connect.Request[v1.GetHealthRequest]) (*connect.Response[v1.GetHealthResponse], error) {
	return connect.NewResponse(&v1.GetHealthResponse{}), nil
}

// GetUpload implements v1connect.StorageServiceHandler.
func (s *StorageService) GetUpload(context.Context, *connect.Request[v1.GetUploadRequest]) (*connect.Response[v1.GetUploadResponse], error) {
	panic("unimplemented")
}

// Ping implements v1connect.StorageServiceHandler.
func (s *StorageService) Ping(context.Context, *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Message: "pong"}), nil
}

// StreamImage implements v1connect.StorageServiceHandler.
func (s *StorageService) StreamImage(context.Context, *connect.Request[v1.StreamImageRequest], *connect.ServerStream[v1.StreamImageResponse]) error {
	panic("unimplemented")
}

// StreamTrack implements v1connect.StorageServiceHandler.
func (s *StorageService) StreamTrack(context.Context, *connect.Request[v1.StreamTrackRequest], *connect.ServerStream[v1.StreamTrackResponse]) error {
	panic("unimplemented")
}

// UploadImage implements v1connect.StorageServiceHandler.
func (s *StorageService) UploadImage(context.Context, *connect.Request[v1.UploadImageRequest]) (*connect.Response[v1.UploadImageResponse], error) {
	panic("unimplemented")
}

// UploadTrack implements v1connect.StorageServiceHandler.
func (s *StorageService) UploadTrack(context.Context, *connect.Request[v1.UploadTrackRequest]) (*connect.Response[v1.UploadTrackResponse], error) {
	panic("unimplemented")
}
