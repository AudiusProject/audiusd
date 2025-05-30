package server

import (
	"fmt"
	"time"

	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
)

const (
	StatusCacheKey = "status"
)

func (s *Server) getStatus() (*v1.GetStatusResponse, error) {
	status, ok := s.statusCache.Get(StatusCacheKey)
	if !ok {
		return nil, fmt.Errorf("status not found")
	}
	return status, nil
}

func (s *Server) startStatusRefresh() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.refreshStatus(); err != nil {
			s.logger.Errorf("error refreshing status: %v", err)
		}
	}
}

func (s *Server) refreshStatus() error {
	return nil

}
