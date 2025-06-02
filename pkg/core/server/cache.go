package server

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/cometbft/cometbft/types"
	"github.com/maypok86/otter"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ProcessStateABCI           = "abci"
	ProcessStateRegistryBridge = "registryBridge"
	ProcessStateEchoServer     = "echoServer"
	ProcessStateSyncTasks      = "syncTasks"
	ProcessStatePeerManager    = "peerManager"
	ProcessStateEthNodeManager = "ethNodeManager"
	ProcessStateCache          = "cache"

	NodeInfoKey      = "nodeInfo"
	ChainInfoKey     = "chainInfo"
	SyncInfoKey      = "syncInfo"
	PruningInfoKey   = "pruningInfo"
	ResourceInfoKey  = "resourceInfo"
	ValidatorInfoKey = "validatorInfo"
	MempoolInfoKey   = "mempoolInfo"
	SnapshotInfoKey  = "snapshotInfo"
)

// a simple in memory cache of frequently queried things
// maybe upgrade to something like bigcache later
type Cache struct {
	// old values, replace with otter cache later
	currentHeight atomic.Int64
	catchingUp    atomic.Bool

	// process states
	abciState           otter.Cache[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo]
	registryBridgeState otter.Cache[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo]
	echoServerState     otter.Cache[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo]
	syncTasksState      otter.Cache[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo]
	peerManagerState    otter.Cache[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo]
	ethNodeManagerState otter.Cache[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo]
	cacheState          otter.Cache[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo]

	// info
	nodeInfo      otter.Cache[string, *v1.GetStatusResponse_NodeInfo]
	chainInfo     otter.Cache[string, *v1.GetStatusResponse_ChainInfo]
	syncInfo      otter.Cache[string, *v1.GetStatusResponse_SyncInfo]
	pruningInfo   otter.Cache[string, *v1.GetStatusResponse_PruningInfo]
	resourceInfo  otter.Cache[string, *v1.GetStatusResponse_ResourceInfo]
	validatorInfo otter.Cache[string, *v1.GetStatusResponse_ValidatorInfo]
	mempoolInfo   otter.Cache[string, *v1.GetStatusResponse_MempoolInfo]
	snapshotInfo  otter.Cache[string, *v1.GetStatusResponse_SnapshotInfo]
}

func NewCache() *Cache {
	c := &Cache{}
	c.currentHeight.Store(0)
	c.catchingUp.Store(true) // assume syncing on startup
	return c
}

func (c *Cache) initCaches() error {
	g := errgroup.Group{}

	g.Go(func() error {
		abciState, err := otter.MustBuilder[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create abci state cache: %v", err)
		}
		c.abciState = abciState
		c.abciState.Set(ProcessStateABCI, &v1.GetStatusResponse_ProcessInfo_ProcessStateInfo{
			State:     v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_STARTING,
			StartedAt: timestamppb.New(time.Now()),
		})
		return nil
	})

	g.Go(func() error {
		registryBridgeState, err := otter.MustBuilder[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create registry bridge state cache: %v", err)
		}
		c.registryBridgeState = registryBridgeState
		c.registryBridgeState.Set(ProcessStateRegistryBridge, &v1.GetStatusResponse_ProcessInfo_ProcessStateInfo{
			State:     v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_STARTING,
			StartedAt: timestamppb.New(time.Now()),
		})
		return nil
	})

	g.Go(func() error {
		echoServerState, err := otter.MustBuilder[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create echo server state cache: %v", err)
		}
		c.echoServerState = echoServerState
		c.echoServerState.Set(ProcessStateEchoServer, &v1.GetStatusResponse_ProcessInfo_ProcessStateInfo{
			State:     v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_STARTING,
			StartedAt: timestamppb.New(time.Now()),
		})
		return nil
	})

	g.Go(func() error {
		syncTasksState, err := otter.MustBuilder[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create sync tasks state cache: %v", err)
		}
		c.syncTasksState = syncTasksState
		c.syncTasksState.Set(ProcessStateSyncTasks, &v1.GetStatusResponse_ProcessInfo_ProcessStateInfo{
			State:     v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_STARTING,
			StartedAt: timestamppb.New(time.Now()),
		})
		return nil
	})

	g.Go(func() error {
		peerManagerState, err := otter.MustBuilder[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create peer manager state cache: %v", err)
		}
		c.peerManagerState = peerManagerState
		c.peerManagerState.Set(ProcessStatePeerManager, &v1.GetStatusResponse_ProcessInfo_ProcessStateInfo{
			State:     v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_STARTING,
			StartedAt: timestamppb.New(time.Now()),
		})
		return nil
	})

	g.Go(func() error {
		ethNodeManagerState, err := otter.MustBuilder[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create eth node manager state cache: %v", err)
		}
		c.ethNodeManagerState = ethNodeManagerState
		c.ethNodeManagerState.Set(ProcessStateEthNodeManager, &v1.GetStatusResponse_ProcessInfo_ProcessStateInfo{
			State:     v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_STARTING,
			StartedAt: timestamppb.New(time.Now()),
		})
		return nil
	})

	g.Go(func() error {
		cacheState, err := otter.MustBuilder[string, *v1.GetStatusResponse_ProcessInfo_ProcessStateInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create cache state cache: %v", err)
		}
		c.cacheState = cacheState
		c.cacheState.Set(ProcessStateCache, &v1.GetStatusResponse_ProcessInfo_ProcessStateInfo{
			State:     v1.GetStatusResponse_ProcessInfo_PROCESS_STATE_STARTING,
			StartedAt: timestamppb.New(time.Now()),
		})
		return nil
	})

	g.Go(func() error {

		nodeInfo, err := otter.MustBuilder[string, *v1.GetStatusResponse_NodeInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create node info cache: %v", err)
		}
		c.nodeInfo = nodeInfo
		return nil
	})

	g.Go(func() error {
		chainInfo, err := otter.MustBuilder[string, *v1.GetStatusResponse_ChainInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create chain info cache: %v", err)
		}
		c.chainInfo = chainInfo
		return nil
	})

	g.Go(func() error {
		syncInfo, err := otter.MustBuilder[string, *v1.GetStatusResponse_SyncInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create sync info cache: %v", err)
		}
		c.syncInfo = syncInfo
		return nil
	})

	g.Go(func() error {
		pruningInfo, err := otter.MustBuilder[string, *v1.GetStatusResponse_PruningInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create pruning info cache: %v", err)
		}
		c.pruningInfo = pruningInfo
		return nil
	})

	g.Go(func() error {
		resourceInfo, err := otter.MustBuilder[string, *v1.GetStatusResponse_ResourceInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create resource info cache: %v", err)
		}
		c.resourceInfo = resourceInfo
		return nil
	})

	g.Go(func() error {
		validatorInfo, err := otter.MustBuilder[string, *v1.GetStatusResponse_ValidatorInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create validator info cache: %v", err)
		}
		c.validatorInfo = validatorInfo
		return nil
	})

	g.Go(func() error {
		mempoolInfo, err := otter.MustBuilder[string, *v1.GetStatusResponse_MempoolInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create mempool info cache: %v", err)
		}
		c.mempoolInfo = mempoolInfo
		return nil
	})

	g.Go(func() error {
		snapshotInfo, err := otter.MustBuilder[string, *v1.GetStatusResponse_SnapshotInfo](1).Build()
		if err != nil {
			return fmt.Errorf("failed to create snapshot info cache: %v", err)
		}
		c.snapshotInfo = snapshotInfo
		return nil
	})

	return g.Wait()
}

// maybe put a separate errgroup in here for things that
// continuously hydrate the cache
func (s *Server) startCache() error {
	if err := s.cache.initCaches(); err != nil {
		return fmt.Errorf("failed to initialize caches: %v", err)
	}

	<-s.awaitRpcReady

	status, err := s.rpc.Status(context.Background())
	if err != nil {
		return fmt.Errorf("could not get initial status: %v", err)
	}

	s.cache.currentHeight.Store(status.SyncInfo.LatestBlockHeight)

	node := s.node
	eb := node.EventBus()

	if eb == nil {
		return errors.New("event bus not ready")
	}

	subscriberID := "block-cache-subscriber"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	query := types.EventQueryNewBlock
	subscription, err := eb.Subscribe(ctx, subscriberID, query)
	if err != nil {
		return fmt.Errorf("failed to subscribe to NewBlock events: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping block event subscription")
			return nil
		case msg := <-subscription.Out():
			blockEvent := msg.Data().(types.EventDataNewBlock)
			blockHeight := blockEvent.Block.Height
			s.cache.currentHeight.Store(blockHeight)
		case err := <-subscription.Canceled():
			s.logger.Errorf("Subscription cancelled: %v", err)
			return nil
		}
	}
}
