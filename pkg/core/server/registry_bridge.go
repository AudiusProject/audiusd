// Keeps the validators updated in cometbft and core up to date with what is present on the ethereum node registry.
package server

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"connectrpc.com/connect"
	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	ethv1 "github.com/AudiusProject/audiusd/pkg/api/eth/v1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/AudiusProject/audiusd/pkg/eth/contracts"
	"github.com/cometbft/cometbft/crypto/ed25519"

	geth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"
)

func (s *Server) startRegistryBridge() error {
	ctx := context.Background()
	ticker := time.NewTicker(5 * time.Second)

ethstatus:
	for {
		select {
		case <-ticker.C:
			if status, err := s.eth.GetStatus(ctx, connect.NewRequest(&ethv1.GetStatusRequest{})); err != nil {
				s.logger.Errorf("error getting eth service status: %v", err)
				continue
			} else if !status.Msg.Ready {
				s.logger.Info("waiting for eth service to be ready")
				continue
			} else {
				break ethstatus
			}
		case <-ctx.Done():
			return errors.New("context canceled")
		}
	}

	if s.isDevEnvironment() {
		s.logger.Info("running in dev, registering on ethereum")
		if err := s.registerSelfOnEth(ctx); err != nil {
			s.logger.Errorf("error registering onto eth: %v", err)
			return err
		}
	}

	close(s.awaitEthReady)
	<-s.awaitRpcReady
	s.logger.Info("starting registry bridge")

	// check comet status
	if _, err := s.rpc.Status(context.Background()); err != nil {
		s.logger.Errorf("init registry bridge failed comet rpc status: %v", err)
		return err
	}

	if err := s.awaitNodeCatchup(context.Background()); err != nil {
		s.logger.Errorf("error awaiting node catchup: %v", err)
		return err
	}

	timeout := time.After(120 * time.Minute)
	delay := 2 * time.Second
	ticker = time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ticker.C:
			if err := s.RegisterSelf(); err != nil {
				s.logger.Errorf("node registration failed, will try again: %v", err)
				delay *= 2
				s.logger.Infof("Retrying registration in %s", delay)
				ticker.Reset(delay)
			} else {
				s.listenForEthContractEvents(context.Background())
				return nil
			}
		case <-timeout:
			s.logger.Warn("exhausted registration retries, continuing unregistered")
			return nil
		case <-ctx.Done():
			return errors.New("context canceled")
		}
	}
	return nil
}

func (s *Server) listenForEthContractEvents(ctx context.Context) {
	deregChan := s.eth.SubscribeToDeregistrationEvents()
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("context canceled, stopping subscription to eth events")
			return
		case dereg := <-deregChan:
			s.logger.Info("received deregistration event")
			// brief, randomized pause to allow deregistration event to propogate
			// to all nodes and prevent thundering herd of deregistration attestations and txs
			rand.Seed(time.Now().UnixNano())
			randInterval := rand.Intn(10) + 10
			time.Sleep(time.Duration(randInterval) * time.Second)
			s.deregisterMissingNode(ctx, dereg.DelegateWallet)
		}
	}
}

// checks mainnet eth for itself, if registered and not
// already in the comet state will register itself on comet
func (s *Server) RegisterSelf() error {
	ctx := context.Background()

	if s.isSelfAlreadyRegistered(ctx) {
		s.logger.Info("Skipping registration, we are already registered.")
		return nil
	}

	nodeEndpoint := s.config.NodeEndpoint

	ep, err := s.eth.GetRegisteredEndpointInfo(
		ctx,
		connect.NewRequest(&ethv1.GetRegisteredEndpointInfoRequest{
			Endpoint: nodeEndpoint,
		}),
	)
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			if connectErr.Code() == connect.CodeNotFound {
				s.logger.Infof("node %s : %s not registered on Ethereum", s.config.WalletAddress, nodeEndpoint)
				s.logger.Info("continuing unregistered")
				return nil
			}
		}
		return fmt.Errorf("could not register self: unexpected error: %w", err)
	}

	nodeRecord, err := s.db.GetNodeByEndpoint(ctx, nodeEndpoint)
	if errors.Is(err, pgx.ErrNoRows) {
		s.logger.Infof("node %s not found on comet but found on eth, registering", nodeEndpoint)
		if err := s.registerSelfOnComet(ctx, geth.HexToAddress(s.config.WalletAddress), big.NewInt(ep.Msg.Se.BlockNumber), fmt.Sprint(ep.Msg.Se.Id)); err != nil {
			return fmt.Errorf("could not register on comet: %v", err)
		}
		return nil
	} else if err != nil {
		return err
	}

	s.logger.Infof("node %s : %s registered on network %s", nodeRecord.EthAddress, nodeRecord.Endpoint, s.config.Environment)
	return nil
}

func (s *Server) isDevEnvironment() bool {
	return s.config.Environment == "dev" || s.config.Environment == "sandbox"
}

func (s *Server) registerSelfOnComet(ctx context.Context, delegateOwnerWallet geth.Address, ethBlock *big.Int, spID string) error {
	if res, err := s.eth.IsDuplicateDelegateWallet(
		ctx,
		connect.NewRequest(&ethv1.IsDuplicateDelegateWalletRequest{Wallet: s.config.WalletAddress}),
	); err != nil {
		return fmt.Errorf("could not check for duplicate delegate wallet: %w", err)
	} else if res.Msg.IsDuplicate {
		s.logger.Errorf("node is a duplicate, not registering on comet: %s", s.config.WalletAddress)
		return nil
	}

	if s.cache.catchingUp.Load() {
		return errors.New("aborting comet registration because node is still syncing")
	}

	genValidators := s.config.GenesisFile.Validators
	isGenValidator := false
	for _, validator := range genValidators {
		if validator.Address.String() == s.config.ProposerAddress {
			isGenValidator = true
			break
		}
	}

	peers := s.GetPeers()
	noPeers := len(peers) == 0

	if !isGenValidator && noPeers {
		return errors.New("not in genesis and no peers, retrying to register on comet later")
	}

	serviceType, err := serviceType(s.config.NodeType)
	if err != nil {
		return fmt.Errorf("invalid node type: %v", err)
	}

	addrs, err := s.db.GetAllEthAddressesOfRegisteredNodes(ctx)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to get all registered nodes: %v", err)
	}
	keyBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(keyBytes, ethBlock.Uint64())
	rendezvous := common.GetAttestorRendezvous(addrs, keyBytes, s.config.AttRegistrationRSize)

	attestations := make([]string, 0, s.config.AttRegistrationRSize)
	reg := &corev1.ValidatorRegistration{
		CometAddress:   s.config.ProposerAddress,
		PubKey:         s.config.CometKey.PubKey().Bytes(),
		Power:          int64(s.config.ValidatorVotingPower),
		DelegateWallet: delegateOwnerWallet.Hex(),
		Endpoint:       s.config.NodeEndpoint,
		NodeType:       common.HexToUtf8(serviceType),
		EthBlock:       ethBlock.Int64(),
		SpId:           spID,
		Deadline:       s.cache.currentHeight.Load() + 120,
	}
	for addr := range rendezvous {
		if peer, ok := peers[addr]; ok {
			resp, err := peer.GetRegistrationAttestation(ctx, connect.NewRequest(&corev1.GetRegistrationAttestationRequest{
				Registration: &corev1.ValidatorRegistration{
					CometAddress:   s.config.ProposerAddress,
					PubKey:         s.config.CometKey.PubKey().Bytes(),
					Power:          int64(s.config.ValidatorVotingPower),
					DelegateWallet: delegateOwnerWallet.Hex(),
					Endpoint:       s.config.NodeEndpoint,
					NodeType:       common.HexToUtf8(serviceType),
					EthBlock:       ethBlock.Int64(),
					SpId:           spID,
					Deadline:       s.cache.currentHeight.Load() + 120,
				},
			}))
			if err != nil {
				s.logger.Errorf("failed to get registration attestation from %s: %v", addr, err)
				continue
			}
			attestations = append(attestations, resp.Msg.Signature)
		}
	}

	registrationAtt := &corev1.Attestation{
		Signatures: attestations,
		Body:       &corev1.Attestation_ValidatorRegistration{ValidatorRegistration: reg},
	}

	txBytes, err := proto.Marshal(registrationAtt)
	if err != nil {
		return fmt.Errorf("failure to marshal register tx: %v", err)
	}

	sig, err := common.EthSign(s.config.EthereumKey, txBytes)
	if err != nil {
		return fmt.Errorf("could not sign register tx: %v", err)
	}

	tx := &corev1.SignedTransaction{
		Signature: sig,
		RequestId: uuid.NewString(),
		Transaction: &corev1.SignedTransaction_Attestation{
			Attestation: registrationAtt,
		},
	}

	txreq := &corev1.SendTransactionRequest{
		Transaction: tx,
	}

	txhash, err := s.self.SendTransaction(context.Background(), connect.NewRequest(txreq))
	if err != nil {
		return fmt.Errorf("send register tx failed: %v", err)
	}

	s.logger.Infof("registered node %s in tx %s", s.config.NodeEndpoint, txhash)

	return nil
}

func (s *Server) awaitNodeCatchup(ctx context.Context) error {
	retries := 60
	for tries := retries; tries >= 0; tries-- {
		res, err := s.rpc.Status(ctx)
		if err != nil {
			s.logger.Errorf("error getting comet health: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if res.SyncInfo.CatchingUp {
			time.Sleep(10 * time.Second)
			continue
		}

		// no health error nor catching up
		return nil
	}
	return errors.New("timeout waiting for comet to catch up")
}

func (s *Server) isSelfAlreadyRegistered(ctx context.Context) bool {
	res, err := s.db.GetNodeByEndpoint(ctx, s.config.NodeEndpoint)

	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}

	if err != nil {
		s.logger.Errorf("error getting registered nodes: %v", err)
		return false
	}

	// return if owner wallets match
	return res.EthAddress == s.config.WalletAddress
}

func (s *Server) IsNodeRegisteredOnEthereum(ctx context.Context, endpoint, delegateWallet string, ethBlock int64) (bool, error) {
	ep, err := s.eth.GetRegisteredEndpointInfo(
		ctx,
		connect.NewRequest(&ethv1.GetRegisteredEndpointInfoRequest{
			Endpoint: endpoint,
		}),
	)
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			if connectErr.Code() == connect.CodeNotFound {
				return false, nil
			}
		}
		return false, fmt.Errorf("could check registration status for node at %s with address %s: %w", endpoint, delegateWallet, err)
	}

	if ep.Msg.Se.BlockNumber != ethBlock || ep.Msg.Se.DelegateWallet != delegateWallet {
		return false, nil
	}
	return true, nil
}

func (s *Server) registerSelfOnEth(ctx context.Context) error {
	if _, err := s.eth.GetRegisteredEndpointInfo(
		context.Background(),
		connect.NewRequest(&ethv1.GetRegisteredEndpointInfoRequest{
			Endpoint: s.config.NodeEndpoint,
		}),
	); err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			if connectErr.Code() == connect.CodeNotFound {
				keyBytes := crypto.FromECDSA(s.config.EthereumKey)
				keyHex := hex.EncodeToString(keyBytes)
				var st string
				switch s.config.NodeType {
				case config.Discovery:
					st = "discovery-node"
				default:
					st = "content-node"
				}

				if _, err := s.eth.Register(
					context.Background(),
					connect.NewRequest(&ethv1.RegisterRequest{
						DelegateKey: keyHex,
						Endpoint:    s.config.NodeEndpoint,
						ServiceType: st,
					}),
				); err != nil {
					s.logger.Errorf("could not register on eth: %v", err)
					return fmt.Errorf("could not register on eth: %v", err)
				}
				return nil
			}
		}
		return fmt.Errorf("could not register self: unexpected error: %v", err)
	}

	// Already registered
	return nil
}

func (s *Server) deregisterMissingNode(ctx context.Context, ethAddress string) {
	node, err := s.db.GetRegisteredNodeByEthAddress(ctx, ethAddress)
	if err != nil {
		s.logger.Error("could not deregister missing node", "address", ethAddress, "error", err)
		return
	}

	addrs, err := s.db.GetAllEthAddressesOfRegisteredNodes(ctx)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.logger.Error("could not deregister node: failed to get currently registered nodes", "address", ethAddress, "error", err)
		return
	}
	pubKey := ed25519.PubKey(node.CometPubKey)
	rendezvous := common.GetAttestorRendezvous(addrs, pubKey.Bytes(), s.config.AttDeregistrationRSize)
	attestations := make([]string, 0, s.config.AttRegistrationRSize)
	dereg := corev1.ValidatorDeregistration{
		CometAddress: node.CometAddress,
		PubKey:       pubKey.Bytes(),
		Deadline:     s.cache.currentHeight.Load() + 120,
	}

	peers := s.GetPeers()
	for addr := range rendezvous {
		if peer, ok := peers[addr]; ok {
			deregCopy := dereg
			resp, err := peer.GetDeregistrationAttestation(ctx, connect.NewRequest(&corev1.GetDeregistrationAttestationRequest{
				Deregistration: &deregCopy,
			}))
			if err != nil {
				s.logger.Error("failed to get deregistration attestation from %s: %v", addr, err)
				continue
			}
			attestations = append(attestations, resp.Msg.Signature)
		}
	}

	deregistrationAtt := &corev1.Attestation{
		Signatures: attestations,
		Body:       &corev1.Attestation_ValidatorDeregistration{ValidatorDeregistration: &dereg},
	}

	txBytes, err := proto.Marshal(deregistrationAtt)
	if err != nil {
		s.logger.Error("failure to marshal deregister tx", "error", err)
		return
	}

	sig, err := common.EthSign(s.config.EthereumKey, txBytes)
	if err != nil {
		s.logger.Error("could not sign deregister tx", "error", err)
		return
	}

	tx := &corev1.SignedTransaction{
		Signature: sig,
		RequestId: uuid.NewString(),
		Transaction: &corev1.SignedTransaction_Attestation{
			Attestation: deregistrationAtt,
		},
	}

	txreq := &corev1.SendTransactionRequest{
		Transaction: tx,
	}

	txhash, err := s.self.SendTransaction(context.Background(), connect.NewRequest(txreq))
	if err != nil {
		s.logger.Error("send deregister tx failed", "error", err)
		return
	}

	s.logger.Infof("deregistered node %s in tx %s", s.config.NodeEndpoint, txhash)
}

func serviceType(nt config.NodeType) ([32]byte, error) {
	switch nt {
	case config.Discovery:
		return contracts.DiscoveryNode, nil
	case config.Content:
		return contracts.ContentNode, nil
	}
	return [32]byte{}, fmt.Errorf("node type provided not valid: %v", nt)
}
