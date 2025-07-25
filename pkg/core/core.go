package core

import (
	"context"
	"fmt"
	_ "net/http/pprof"

	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/AudiusProject/audiusd/pkg/core/console"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/AudiusProject/audiusd/pkg/core/server"
	"github.com/AudiusProject/audiusd/pkg/eth"
	"github.com/AudiusProject/audiusd/pkg/lifecycle"
	"github.com/AudiusProject/audiusd/pkg/pos"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(ctx context.Context, lc *lifecycle.Lifecycle, logger *common.Logger, posChannel chan pos.PoSRequest, coreService *server.CoreService, ethService *eth.EthService) error {
	return run(ctx, lc, logger, posChannel, coreService, ethService)
}

func run(ctx context.Context, lc *lifecycle.Lifecycle, logger *common.Logger, posChannel chan pos.PoSRequest, coreService *server.CoreService, ethService *eth.EthService) error {
	logger.Info("good morning!")

	config, cometConfig, err := config.SetupNode(logger)
	if err != nil {
		return fmt.Errorf("setting up node: %v", err)
	}

	logger.Info("configuration created")

	// db migrations
	if err := db.RunMigrations(logger, config.PSQLConn, config.RunDownMigrations()); err != nil {
		return fmt.Errorf("running migrations: %v", err)
	}

	logger.Info("db migrations successful")

	// Use the passed context for the pool
	pool, err := pgxpool.New(ctx, config.PSQLConn)
	if err != nil {
		return fmt.Errorf("couldn't create pgx pool: %v", err)
	}
	defer pool.Close()

	s, err := server.NewServer(lc, config, cometConfig, logger, pool, ethService, posChannel)
	if err != nil {
		return fmt.Errorf("server init error: %v", err)
	}

	s.CompactStateDB()
	s.CompactBlockstoreDB()
	logger.Info("finished compacting db")

	// console gets run from core(main).go since it is an isolated go module
	// unlike the other modules that register themselves on the echo http server
	if config.ConsoleModule {
		e := s.GetEcho()
		con, err := console.NewConsole(config, logger, e, pool)
		if err != nil {
			logger.Errorf("console init error: %v", err)
			return err
		}
		go func() {
			logger.Info("core console starting")
			if err := con.Start(); err != nil {
				logger.Errorf("console couldn't start or crashed: %v", err)
				return
			}
		}()
	}

	// create core service
	coreService.SetCore(s)

	if err := s.Start(); err != nil {
		logger.Errorf("something crashed: %v", err)
		return err
	}

	return s.Shutdown()
}
