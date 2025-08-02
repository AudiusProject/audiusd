package eth

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	v1 "github.com/AudiusProject/audiusd/pkg/api/eth/v1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/eth/contracts"
	"github.com/AudiusProject/audiusd/pkg/eth/contracts/gen"
	"github.com/AudiusProject/audiusd/pkg/eth/db"
	"github.com/AudiusProject/audiusd/pkg/pubsub"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DeregistrationTopic = "deregistration-subscriber"
)

type DeregistrationPubsub = pubsub.Pubsub[*v1.ServiceEndpoint]

type EthService struct {
	rpcURL          string
	dbURL           string
	registryAddress string
	env             string

	rpc         *ethclient.Client
	db          *db.Queries
	pool        *pgxpool.Pool
	logger      *common.Logger
	c           *contracts.AudiusContracts
	deregPubsub *DeregistrationPubsub

	isReady atomic.Bool
}

func NewEthService(dbURL, rpcURL, registryAddress string, logger *common.Logger, environment string) *EthService {
	return &EthService{
		logger:          logger.Child("eth"),
		rpcURL:          rpcURL,
		dbURL:           dbURL,
		registryAddress: registryAddress,
		env:             environment,
	}
}

func (eth *EthService) Run(ctx context.Context) error {
	// Init db
	if eth.dbURL == "" {
		return fmt.Errorf("dbUrl environment variable not set")
	}

	if err := db.RunMigrations(eth.logger, eth.dbURL, false); err != nil {
		return fmt.Errorf("error running migrations: %v", err)
	}

	pgConfig, err := pgxpool.ParseConfig(eth.dbURL)
	if err != nil {
		return fmt.Errorf("error parsing database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return fmt.Errorf("error creating database pool: %v", err)
	}
	eth.pool = pool
	eth.db = db.New(pool)

	// Init pubsub
	eth.deregPubsub = pubsub.NewPubsub[*v1.ServiceEndpoint]()

	// Init eth rpc
	wsRpcUrl := eth.rpcURL
	if strings.HasPrefix(eth.rpcURL, "https") {
		wsRpcUrl = "wss" + strings.TrimPrefix(eth.rpcURL, "https")
	} else if strings.HasPrefix(eth.rpcURL, "http:") { // local devnet
		wsRpcUrl = "ws" + strings.TrimPrefix(eth.rpcURL, "http")
	}
	ethrpc, err := ethclient.Dial(wsRpcUrl)
	if err != nil {
		return fmt.Errorf("eth client dial err: %v", err)
	}
	eth.rpc = ethrpc
	defer ethrpc.Close()

	// Init contracts
	c, err := contracts.NewAudiusContracts(eth.rpc, eth.registryAddress)
	if err != nil {
		return fmt.Errorf("failed to initialize eth contracts: %v", err)
	}
	eth.c = c

	eth.logger.Infof("starting eth service")

	if err := eth.startEthDataManager(ctx); err != nil {
		return fmt.Errorf("Error running endpoint manager: %w", err)
	}

	return nil
}

func (eth *EthService) startEthDataManager(ctx context.Context) error {
	// hydrate eth data at startup
	delay := 2 * time.Second
	ticker := time.NewTicker(delay)
initial:
	for {
		select {
		case <-ticker.C:
			if err := eth.hydrateEthData(ctx); err != nil {
				eth.logger.Errorf("error gathering registered eth endpoints: %v", err)
				delay *= 2
				eth.logger.Infof("retrying in %s seconds", delay)
				ticker.Reset(delay)
			} else {
				break initial
			}
		case <-ctx.Done():
			return errors.New("context canceled")
		}
	}

	eth.isReady.Store(true)

	// Instantiate the contract
	spf, err := eth.c.GetServiceProviderFactoryContract()
	if err != nil {
		return fmt.Errorf("failed to bind service provider factory contract: %v", err)
	}

	watchOpts := &bind.WatchOpts{Context: ctx}

	registerChan := make(chan *gen.ServiceProviderFactoryRegisteredServiceProvider)
	deregisterChan := make(chan *gen.ServiceProviderFactoryDeregisteredServiceProvider)
	updateChan := make(chan *gen.ServiceProviderFactoryEndpointUpdated)

	registerSub, err := spf.WatchRegisteredServiceProvider(watchOpts, registerChan, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to endpoint registration events: %v", err)
	}

	deregisterSub, err := spf.WatchDeregisteredServiceProvider(watchOpts, deregisterChan, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to endpoint deregistration events: %v", err)
	}

	updateSub, err := spf.WatchEndpointUpdated(watchOpts, updateChan, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to endpoint update events: %v", err)
	}

	ticker = time.NewTicker(1 * time.Hour)

	for {
		select {
		case err := <-registerSub.Err():
			return fmt.Errorf("register event subscription error: %v", err)
		case err := <-deregisterSub.Err():
			return fmt.Errorf("deregister event subscription error: %v", err)
		case err := <-updateSub.Err():
			return fmt.Errorf("update event subscription error: %v", err)
		case reg := <-registerChan:
			if err := eth.addRegisteredEndpoint(ctx, reg.SpID, reg.ServiceType, reg.Endpoint, reg.Owner); err != nil {
				eth.logger.Error("could not handle registration event: %v", err)
				continue
			}
			if err := eth.updateServiceProvider(ctx, reg.Owner); err != nil {
				eth.logger.Error("could not update service provider from registration event: %v", err)
				continue
			}
		case dereg := <-deregisterChan:
			if err := eth.deleteAndDeregisterEndpoint(ctx, dereg.SpID, dereg.ServiceType, dereg.Endpoint, dereg.Owner); err != nil {
				eth.logger.Error("could not handle deregistration event: %v", err)
				continue
			}
			if err := eth.updateServiceProvider(ctx, dereg.Owner); err != nil {
				eth.logger.Error("could not update service provider from deregistration event: %v", err)
				continue
			}
		case update := <-updateChan:
			if err := eth.deleteAndDeregisterEndpoint(ctx, update.SpID, update.ServiceType, update.OldEndpoint, update.Owner); err != nil {
				eth.logger.Error("could not handle deregistration phase of update event: %v", err)
				continue
			}
			if err := eth.addRegisteredEndpoint(ctx, update.SpID, update.ServiceType, update.NewEndpoint, update.Owner); err != nil {
				eth.logger.Error("could not handle registration phase of update event: %v", err)
				continue
			}
			if err := eth.updateServiceProvider(ctx, update.Owner); err != nil {
				eth.logger.Error("could not update service provider from update event: %v", err)
				continue
			}
		case <-ticker.C:
			if err := eth.hydrateEthData(ctx); err != nil {
				// crash if periodic updates fail - it may be necessary to reestablish connections
				return fmt.Errorf("error gathering eth endpoints: %v", err)
			}
		case <-ctx.Done():
			return errors.New("context canceled")
		}
	}

	return nil
}

func (eth *EthService) SubscribeToDeregistrationEvents() chan *v1.ServiceEndpoint {
	return eth.deregPubsub.Subscribe(DeregistrationTopic, 10)
}

func (eth *EthService) UnsubscribeFromDeregistrationEvents(ch chan *v1.ServiceEndpoint) {
	eth.deregPubsub.Unsubscribe(DeregistrationTopic, ch)
}

func (eth *EthService) deleteAndDeregisterEndpoint(ctx context.Context, spID *big.Int, serviceType [32]byte, endpoint string, owner ethcommon.Address) error {
	st, err := contracts.ServiceTypeToString(serviceType)
	if err != nil {
		return err
	}
	ep, err := eth.db.GetRegisteredEndpoint(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("could not fetch endpoint %s from db: %v", endpoint, err)
	}
	if err := eth.db.DeleteRegisteredEndpoint(
		ctx,
		db.DeleteRegisteredEndpointParams{
			ID:          int32(spID.Int64()),
			Endpoint:    endpoint,
			Owner:       owner.Hex(),
			ServiceType: st,
		},
	); err != nil {
		return err
	}
	eth.deregPubsub.Publish(
		ctx,
		DeregistrationTopic,
		&v1.ServiceEndpoint{
			Id:             spID.Int64(),
			Owner:          owner.Hex(),
			Endpoint:       endpoint,
			DelegateWallet: ep.DelegateWallet,
		},
	)
	return nil
}

func (eth *EthService) addRegisteredEndpoint(ctx context.Context, spID *big.Int, serviceType [32]byte, endpoint string, owner ethcommon.Address) error {
	st, err := contracts.ServiceTypeToString(serviceType)
	if err != nil {
		return err
	}
	node, err := eth.c.GetRegisteredNode(ctx, spID, serviceType)
	if err != nil {
		return err
	}
	return eth.db.InsertRegisteredEndpoint(
		ctx,
		db.InsertRegisteredEndpointParams{
			ID:             int32(spID.Int64()),
			ServiceType:    st,
			Owner:          owner.Hex(),
			DelegateWallet: node.DelegateOwnerWallet.Hex(),
			Endpoint:       endpoint,
			Blocknumber:    node.BlockNumber.Int64(),
		},
	)
}

func (eth *EthService) updateServiceProvider(ctx context.Context, serviceProviderAddress ethcommon.Address) error {
	serviceProviderFactory, err := eth.c.GetServiceProviderFactoryContract()
	if err != nil {
		return fmt.Errorf("failed to bind service provider factory contract while updating service provider: %v", err)
	}
	opts := &bind.CallOpts{Context: ctx}

	spDetails, err := serviceProviderFactory.GetServiceProviderDetails(opts, serviceProviderAddress)
	if err != nil {
		return fmt.Errorf("failed get service provider details for address %s: %v", serviceProviderAddress.Hex(), err)
	}
	if err := eth.db.InsertServiceProvider(
		ctx,
		db.InsertServiceProviderParams{
			Address:           serviceProviderAddress.Hex(),
			DeployerStake:     spDetails.DeployerStake.Int64(),
			DeployerCut:       spDetails.DeployerCut.Int64(),
			ValidBounds:       spDetails.ValidBounds,
			NumberOfEndpoints: int32(spDetails.NumberOfEndpoints.Int64()),
			MinAccountStake:   spDetails.MinAccountStake.Int64(),
			MaxAccountStake:   spDetails.MaxAccountStake.Int64(),
		},
	); err != nil {
		return fmt.Errorf("could not upsert service provider into eth service db: %v", err)
	}
	return nil
}

func (eth *EthService) hydrateEthData(ctx context.Context) error {
	eth.logger.Info("refreshing eth data")

	nodes, err := eth.c.GetAllRegisteredNodes(ctx)
	if err != nil {
		return fmt.Errorf("could not get registered nodes from contracts: %w", err)
	}

	tx, err := eth.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("could not begin db tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	txq := eth.db.WithTx(tx)

	if err := txq.ClearRegisteredEndpoints(ctx); err != nil {
		return fmt.Errorf("could not clear registered endpoints: %w", err)
	}

	allServiceProviders := make(map[string]*db.EthServiceProvider, len(nodes))
	serviceProviderFactory, err := eth.c.GetServiceProviderFactoryContract()
	if err != nil {
		return fmt.Errorf("failed to bind service provider factory contract: %v", err)
	}
	opts := &bind.CallOpts{Context: ctx}

	for _, node := range nodes {
		st, err := contracts.ServiceTypeToString(node.Type)
		if err != nil {
			return fmt.Errorf("could resolve service type for node: %w", err)
		}
		if err := txq.InsertRegisteredEndpoint(
			ctx,
			db.InsertRegisteredEndpointParams{
				ID:             int32(node.Id.Int64()),
				ServiceType:    st,
				Owner:          node.Owner.Hex(),
				DelegateWallet: node.DelegateOwnerWallet.Hex(),
				Endpoint:       node.Endpoint,
				Blocknumber:    node.BlockNumber.Int64(),
			},
		); err != nil {
			return fmt.Errorf("could not insert registered endpoint into eth indexer db: %w", err)
		}

		if _, ok := allServiceProviders[node.Owner.Hex()]; !ok {
			spDetails, err := serviceProviderFactory.GetServiceProviderDetails(opts, node.Owner)
			if err != nil {
				return fmt.Errorf("failed get service provider details for address %s: %v", node.Owner.Hex(), err)
			}
			allServiceProviders[node.Owner.Hex()] = &db.EthServiceProvider{
				Address:           node.Owner.Hex(),
				DeployerStake:     spDetails.DeployerStake.Int64(),
				DeployerCut:       spDetails.DeployerCut.Int64(),
				ValidBounds:       spDetails.ValidBounds,
				NumberOfEndpoints: int32(spDetails.NumberOfEndpoints.Int64()),
				MinAccountStake:   spDetails.MinAccountStake.Int64(),
				MaxAccountStake:   spDetails.MaxAccountStake.Int64(),
			}
		}
	}

	eth.logger.Info("***** DELETEME Gonna insert sps", "sps", allServiceProviders)
	for _, sp := range allServiceProviders {
		if err := txq.InsertServiceProvider(
			ctx,
			db.InsertServiceProviderParams{
				Address:           sp.Address,
				DeployerStake:     sp.DeployerStake,
				DeployerCut:       sp.DeployerCut,
				ValidBounds:       sp.ValidBounds,
				NumberOfEndpoints: sp.NumberOfEndpoints,
				MinAccountStake:   sp.MinAccountStake,
				MaxAccountStake:   sp.MaxAccountStake,
			},
		); err != nil {
			return fmt.Errorf("could not insert service provider into eth indexer db: %w", err)
		}
	}

	return tx.Commit(ctx)
}
