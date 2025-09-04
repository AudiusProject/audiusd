package server

import (
	"context"
	"errors"

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

// Ping implements v1connect.StorageServiceHandler.
func (s *StorageService) Ping(context.Context, *connect.Request[v1.PingRequest]) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Message: "pong"}), nil
}

// GetIPData implements v1connect.StorageServiceHandler.
func (s *StorageService) GetIPData(ctx context.Context, req *connect.Request[v1.GetIPDataRequest]) (*connect.Response[v1.GetIPDataResponse], error) {
	ip := req.Msg.Ip
	if ip == "" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("must send IP"))
	}

	ipData, err := s.mediorum.getGeoFromIP(ip)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	res := &v1.GetIPDataResponse{
		Country:     ipData.Country,
		CountryCode: ipData.CountryCode,
		Region:      ipData.Region,
		RegionCode:  ipData.RegionCode,
		City:        ipData.City,
		Latitude:    float32(ipData.Latitude),
		Longitude:   float32(ipData.Longitude),
	}
	return connect.NewResponse(res), nil
}
