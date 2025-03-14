package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/AudiusProject/audiusd/pkg/core/contracts"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	"github.com/AudiusProject/audiusd/pkg/core/sdk"
	"github.com/AudiusProject/audiusd/pkg/pos"
	cconfig "github.com/cometbft/cometbft/config"
	nm "github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/rpc/client/local"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type Server struct {
	config         *config.Config
	cometbftConfig *cconfig.Config
	logger         *common.Logger

	httpServer         *echo.Echo
	grpcServer         *grpc.Server
	pool               *pgxpool.Pool
	contracts          *contracts.AudiusContracts
	mediorumPoSChannel chan pos.PoSRequest

	db    *db.Queries
	eth   *ethclient.Client
	node  *nm.Node
	rpc   *local.Local
	mempl *Mempool

	peers   map[string]*sdk.Sdk
	peersMU sync.RWMutex

	txPubsub *TransactionHashPubsub

	cache     *Cache
	abciState *ABCIState

	core_proto.UnimplementedProtocolServer

	ethNodes          []*contracts.Node
	duplicateEthNodes []*contracts.Node
	missingEthNodes   []string
	ethNodeMU         sync.RWMutex

	awaitHttpServerReady chan struct{}
	awaitGrpcServerReady chan struct{}
	awaitRpcReady        chan struct{}
	awaitEthNodesReady   chan struct{}
}

func NewServer(config *config.Config, cconfig *cconfig.Config, logger *common.Logger, pool *pgxpool.Pool, eth *ethclient.Client, posChannel chan pos.PoSRequest) (*Server, error) {
	// create mempool
	mempl := NewMempool(logger, config, db.New(pool), cconfig.Mempool.Size)

	// create pubsubs
	txPubsub := NewPubsub[struct{}]()

	// create contracts
	c, err := contracts.NewAudiusContracts(eth, config.EthRegistryAddress)
	if err != nil {
		return nil, fmt.Errorf("contracts init error: %v", err)
	}

	httpServer := echo.New()
	grpcServer := grpc.NewServer()

	ethNodes := []*contracts.Node{}
	duplicateEthNodes := []*contracts.Node{}

	s := &Server{
		config:         config,
		cometbftConfig: cconfig,
		logger:         logger.Child("server"),

		pool:               pool,
		contracts:          c,
		mediorumPoSChannel: posChannel,

		db:        db.New(pool),
		eth:       eth,
		mempl:     mempl,
		peers:     make(map[string]*sdk.Sdk),
		txPubsub:  txPubsub,
		cache:     NewCache(),
		abciState: NewABCIState(config.RetainHeight),

		httpServer: httpServer,
		grpcServer: grpcServer,

		ethNodes:          ethNodes,
		duplicateEthNodes: duplicateEthNodes,
		missingEthNodes:   []string{},

		awaitHttpServerReady: make(chan struct{}),
		awaitGrpcServerReady: make(chan struct{}),
		awaitRpcReady:        make(chan struct{}),
		awaitEthNodesReady:   make(chan struct{}),
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	g, _ := errgroup.WithContext(ctx)

	g.Go(s.startABCI)
	g.Go(s.startGRPC)
	g.Go(s.startRegistryBridge)
	g.Go(s.startEchoServer)
	g.Go(s.startSyncTasks)
	g.Go(s.startPeerManager)
	g.Go(s.startEthNodeManager)
	g.Go(s.startCache)
	g.Go(s.startDataCompanion)

	s.logger.Info("services started")

	return g.Wait()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down all services...")

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error { return s.httpServer.Shutdown(ctx) })
	g.Go(s.node.Stop)
	g.Go(func() error {
		s.grpcServer.GracefulStop()
		return nil
	})

	return g.Wait()
}
