package server

import (
	"context"
	"errors"
	"time"
)

var (
	ErrRpcStatusNotFound      = errors.New("local rpc status not returned")
	ErrRpcNotSynced           = errors.New("local rpc not synced")
	ErrCreateValidatorClients = errors.New("couldn't create validator clients")
)

// tasks that execute once the node is fully synced
func (s *Server) startSyncTasks(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			if err := s.onSyncTick(ctx); err != nil {
				s.logger.Debugf("still syncing: %v", err)
			} else {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Server) onSyncTick(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.awaitRpcReady:
	}

	status, _ := s.rpc.Status(ctx)
	if status == nil {
		return ErrRpcStatusNotFound
	}

	if status.SyncInfo.CatchingUp {
		s.cache.catchingUp.Store(true)
		return ErrRpcNotSynced
	} else {
		s.cache.catchingUp.Store(false)
	}

	return nil
}
