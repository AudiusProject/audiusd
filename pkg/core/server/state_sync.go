package server

import "context"

func (s *Server) startStateSync() error {
	if !s.createSnapshot() {
		return nil
	}

	return nil
}

func (s *Server) createSnapshot() bool {
	if s.rpc == nil {
		return false
	}

	status, err := s.rpc.Status(context.Background())
	if err != nil {
		return false
	}

	if status.SyncInfo.LatestBlockHeight == 0 {
		return false
	}

	if status.SyncInfo.CatchingUp {
		return false
	}

	return true
}
