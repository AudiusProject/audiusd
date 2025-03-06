package server

import (
	"context"
	"fmt"
	"sync/atomic"
)

// a simple in memory cache of frequently queried things
// maybe upgrade to something like bigcache later
type Cache struct {
	currentHeight int64
}

func NewCache() *Cache {
	return &Cache{}
}

// maybe put a separate errgroup in here for things that
// continuously hydrate the cache
func (s *Server) startCache() error {
	<-s.awaitRpcReady

	status, err := s.rpc.Status(context.Background())
	if err != nil {
		return fmt.Errorf("could not get initial status: %v", err)
	}

	atomic.StoreInt64(&s.cache.currentHeight, status.SyncInfo.LatestBlockHeight)
	return nil
}
