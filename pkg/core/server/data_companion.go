package server

import (
	"context"
	"fmt"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/cometbft/cometbft/rpc/grpc/client/privileged"
)

func (s *Server) startDataCompanion() error {
	if s.config.Archive {
		return nil
	}

	s.logger.Info("starting data companion")

	<-s.awaitRpcReady

	ctx := context.Background()

	conn, err := privileged.New(ctx, config.PrivilegedServiceSocket, privileged.WithPruningServiceEnabled(true), privileged.WithInsecure())
	if err != nil {
		return fmt.Errorf("dc could not create privileged rpc connection: %v", err)
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		blockRetainHeight, err := conn.GetBlockRetainHeight(ctx)
		if err != nil {
			s.logger.Errorf("dc could not get block retain height: %v", err)
			continue
		}

		s.logger.Infof("dc app retain height: %d block retain height: %d", blockRetainHeight.App, blockRetainHeight.PruningService)

		blockResultsRetainHeight, err := conn.GetBlockResultsRetainHeight(ctx)
		if err != nil {
			s.logger.Errorf("dc could not get block results retain height: %v", err)
			continue
		}

		s.logger.Infof("dc block results retain height: %d", blockResultsRetainHeight)

		retainHeight := s.cache.currentHeight - s.config.RetainHeight
		if retainHeight < 1 {
			retainHeight = 1
		}

		if err := conn.SetBlockRetainHeight(ctx, uint64(retainHeight)); err != nil {
			s.logger.Errorf("dc could not set block retain height: %v", err)
		}

		if err := conn.SetBlockResultsRetainHeight(ctx, uint64(retainHeight)); err != nil {
			s.logger.Errorf("dc could not set block results retain height: %v", err)
		}
	}

	return nil
}
