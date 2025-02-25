// Peers that core is aware of and uses. This is different than the lower level p2p list that cometbft manages.
// This is where we store sdk clients for other validators for the purposes of forwarding transactions, querying health checks, and
// anything else.
package server

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/contracts"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/AudiusProject/audiusd/pkg/core/sdk"
	rpcclient "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/labstack/echo/v4"
	"github.com/serialx/hashring"
)

var legacyDiscoveryProviderProfile = []string{".audius.co", ".creatorseed.com", "dn1.monophonic.digital", ".figment.io", ".tikilabs.com"}

type RegisteredNodeVerboseResponse struct {
	Owner               string `json:"owner"`
	Endpoint            string `json:"endpoint"`
	SpID                uint64 `json:"spID"`
	NodeType            string `json:"type"`
	BlockNumber         uint64 `json:"blockNumber"`
	DelegateOwnerWallet string `json:"delegateOwnerWallet"`
	CometAddress        string `json:"cometAddress"`
}

type RegisteredNodesVerboseResponse struct {
	RegisteredNodes []*RegisteredNodeVerboseResponse `json:"data"`
}

type RegisteredNodesEndpointResponse struct {
	RegisteredNodes []string `json:"data"`
}

type PeerInfo struct {
	endpoint string
	ip       string
	p2pOpen  bool
	nodeID   string
}

func (s *Server) startPeerManager() error {
	<-s.awaitRpcReady

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			if err := s.updateCorePeers(); err != nil {
				s.logger.Errorf("error connecting to core peers: %v", err)
			}
		}()

		go func() {
			defer wg.Done()
			if err := s.updateCometPeers(); err != nil {
				s.logger.Errorf("error connecting to comet peers: %v", err)
			}
		}()

		wg.Wait()
	}

	return nil
}

func (s *Server) updateCorePeers() error {
	validators, err := s.db.GetAllRegisteredNodes(context.Background())
	if err != nil {
		return fmt.Errorf("could not get validators from db: %v", err)
	}

	peers := s.GetPeers()
	addedNewPeer := false
	self := s.config.WalletAddress

	var wg sync.WaitGroup
	wg.Add(len(validators))

	var localPeerMU sync.RWMutex
	for _, validator := range validators {
		go func() {
			defer wg.Done()

			ethaddr := validator.EthAddress
			if ethaddr == self {
				return
			}

			localPeerMU.RLock()
			_, peered := peers[ethaddr]
			localPeerMU.RUnlock()
			if peered {
				return
			}

			parsedURL, err := url.Parse(validator.Endpoint)
			if err != nil {
				s.logger.Errorf("could not parse url for %s: %v", validator.Endpoint, err)
				return
			}

			oapiendpoint := parsedURL.Host
			// don't retry because ticker will handle it
			sdk, err := sdk.NewSdk(sdk.WithOapiendpoint(oapiendpoint), sdk.WithRetries(0), sdk.WithUsehttps(s.config.UseHttpsForSdk))
			if err != nil {
				s.logger.Errorf("could not init sdk for peer %s: %v", oapiendpoint, err)
				return
			}

			// add to peers copy
			localPeerMU.Lock()
			peers[ethaddr] = sdk
			localPeerMU.Unlock()
			if !addedNewPeer {
				addedNewPeer = true
			}
		}()
	}

	wg.Wait()

	if addedNewPeer {
		s.UpdatePeers(peers)
	}

	return nil
}

// UpdatePeers updates the peers map
func (s *Server) UpdatePeers(newPeers map[string]*sdk.Sdk) {
	s.peersMU.Lock()
	defer s.peersMU.Unlock()
	s.peers = newPeers
}

// GetPeers retrieves a snapshot of the current peers map
func (s *Server) GetPeers() map[string]*sdk.Sdk {
	s.peersMU.RLock()
	defer s.peersMU.RUnlock()
	// Return a copy to avoid race conditions
	peersCopy := make(map[string]*sdk.Sdk, len(s.peers))
	for k, v := range s.peers {
		peersCopy[k] = v
	}
	return peersCopy
}

func (s *Server) getRegisteredNodes(c echo.Context) error {
	ctx := c.Request().Context()
	queries := s.db

	path := c.Path()

	discoveryQuery := strings.Contains(path, "discovery")
	contentQuery := strings.Contains(path, "content")
	allQuery := !discoveryQuery && !contentQuery

	verbose := strings.Contains(path, "verbose")

	nodes := []*RegisteredNodeVerboseResponse{}

	if allQuery {
		res, err := queries.GetAllRegisteredNodes(ctx)
		if err != nil {
			return fmt.Errorf("could not get all nodes: %v", err)
		}
		for _, node := range res {
			spID, err := strconv.ParseUint(node.SpID, 10, 32)
			if err != nil {
				return fmt.Errorf("could not convert spid to int: %v", err)
			}

			ethBlock, err := strconv.ParseUint(node.EthBlock, 10, 32)
			if err != nil {
				return fmt.Errorf("could not convert ethblock to int: %v", err)
			}

			nodes = append(nodes, &RegisteredNodeVerboseResponse{
				// TODO: fix this
				Owner:               node.EthAddress,
				Endpoint:            node.Endpoint,
				SpID:                spID,
				NodeType:            node.NodeType,
				BlockNumber:         ethBlock,
				DelegateOwnerWallet: node.EthAddress,
				CometAddress:        node.CometAddress,
			})
		}
	}

	if discoveryQuery {
		res, err := queries.GetRegisteredNodesByType(ctx, common.HexToUtf8(contracts.DiscoveryNode))
		if err != nil {
			return fmt.Errorf("could not get discovery nodes: %v", err)
		}
		for _, node := range res {
			isProd := s.config.Environment == "prod"
			if isProd {
				nodeFound := false
				for _, nodeType := range legacyDiscoveryProviderProfile {
					if nodeFound {
						break
					}
					if strings.Contains(node.Endpoint, nodeType) {
						nodeFound = true
						break
					}
				}
				if !nodeFound {
					continue
				}
			}

			spID, err := strconv.ParseUint(node.SpID, 10, 32)
			if err != nil {
				return fmt.Errorf("could not convert spid to int: %v", err)
			}

			ethBlock, err := strconv.ParseUint(node.EthBlock, 10, 32)
			if err != nil {
				return fmt.Errorf("could not convert ethblock to int: %v", err)
			}

			nodeResponse := &RegisteredNodeVerboseResponse{
				Owner:               node.EthAddress,
				Endpoint:            node.Endpoint,
				SpID:                spID,
				NodeType:            node.NodeType,
				BlockNumber:         ethBlock,
				DelegateOwnerWallet: node.EthAddress,
				CometAddress:        node.CometAddress,
			}

			nodes = append(nodes, nodeResponse)
		}
	}

	if contentQuery {
		res, err := queries.GetRegisteredNodesByType(ctx, common.HexToUtf8(contracts.ContentNode))
		if err != nil {
			return fmt.Errorf("could not get discovery nodes: %v", err)
		}
		for _, node := range res {
			spID, err := strconv.ParseUint(node.SpID, 10, 32)
			if err != nil {
				return fmt.Errorf("could not convert spid to int: %v", err)
			}

			ethBlock, err := strconv.ParseUint(node.EthBlock, 10, 32)
			if err != nil {
				return fmt.Errorf("could not convert ethblock to int: %v", err)
			}

			nodes = append(nodes, &RegisteredNodeVerboseResponse{
				// TODO: fix this
				Owner:               node.EthAddress,
				Endpoint:            node.Endpoint,
				SpID:                spID,
				NodeType:            node.NodeType,
				BlockNumber:         ethBlock,
				DelegateOwnerWallet: node.EthAddress,
				CometAddress:        node.CometAddress,
			})
		}
	}

	if verbose {
		res := RegisteredNodesVerboseResponse{
			RegisteredNodes: nodes,
		}
		return c.JSON(200, res)
	}

	endpoint := []string{}

	for _, node := range nodes {
		endpoint = append(endpoint, node.Endpoint)
	}

	res := RegisteredNodesEndpointResponse{
		RegisteredNodes: endpoint,
	}

	return c.JSON(200, res)
}

func (s *Server) collectCometPeers(nodeEndpoints []string) ([]string, error) {
	ctx := context.Background()
	cometPeers := []string{}
	// nodeID -> PeerInfo
	peerInfo := make(map[string]PeerInfo)

	for _, endpoint := range nodeEndpoints {
		endpoint := fmt.Sprintf("%s/core/debug/comet", endpoint)
		client, err := rpcclient.NewWithTimeout(endpoint, 3)
		if err != nil {
			s.logger.Errorf("could not get remote client: %v", err)
			continue
		}

		netInfo, err := client.NetInfo(ctx)
		if err != nil {
			s.logger.Errorf("could not get net info: %v", err)
			continue
		}

		nodePeers := netInfo.Peers
		s.logger.Infof("node %s has %d peers", endpoint, len(nodePeers))

		for _, peer := range nodePeers {
			nodeID := string(peer.NodeInfo.DefaultNodeID)
			if _, ok := peerInfo[nodeID]; ok {
				s.logger.Infof("node %s already in peerInfo", nodeID)
				continue
			}

			p2pOpen := true
			address := fmt.Sprintf("%s:%d", peer.RemoteIP, 26656)
			conn, err := net.DialTimeout("tcp", address, 3*time.Second)
			if err != nil {
				p2pOpen = false
				s.logger.Warningf("port 26656 on node %s unreachable\n", endpoint)
			}
			if conn != nil {
				conn.Close()
			}

			peerInfo[nodeID] = PeerInfo{
				nodeID:  nodeID,
				ip:      peer.RemoteIP,
				p2pOpen: p2pOpen,
			}

			s.logger.Infof("node %s p2p open: %v", nodeID, p2pOpen)
		}
	}

	return cometPeers, nil
}

// checkPeerCount checks if we have enough peers. Returns true if we have sufficient peers.
func (s *Server) checkPeerCount(ctx context.Context) (bool, error) {
	netInfo, err := s.rpc.NetInfo(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get net info: %v", err)
	}

	return netInfo.NPeers >= 40, nil
}

// getValidatorPeers retrieves all registered validator nodes except ourselves
func (s *Server) getValidatorPeers(ctx context.Context) (map[string]string, []string, error) {
	validators, err := s.db.GetAllRegisteredNodes(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get registered nodes: %v", err)
	}

	validatorIDToEndpoint := make(map[string]string)
	validatorIDs := []string{}

	me := s.config.ProposerAddress
	for _, validator := range validators {
		if validator.CometAddress == me {
			continue
		}
		cometAddress := strings.ToLower(validator.CometAddress)
		validatorIDToEndpoint[cometAddress] = validator.Endpoint
		validatorIDs = append(validatorIDs, cometAddress)
	}

	return validatorIDToEndpoint, validatorIDs, nil
}

// gatherPeerInfo collects peer information from a single validator
func (s *Server) gatherPeerInfo(ctx context.Context, validator db.CoreValidator, nodeInfo *map[string]PeerInfo, nodeInfoMU *sync.Mutex) {
	endpoint := fmt.Sprintf("%s/core/debug/comet", validator.Endpoint)
	client, err := rpcclient.New(endpoint)
	if err != nil {
		s.logger.Errorf("could not get remote client: %v", err)
		return
	}

	netInfo, err := client.NetInfo(ctx)
	if err != nil {
		s.logger.Errorf("could not get net info: %v", err)
		return
	}

	for _, peer := range netInfo.Peers {
		peerID := string(peer.NodeInfo.DefaultNodeID)
		nodeInfoMU.Lock()
		if _, ok := (*nodeInfo)[peerID]; ok {
			nodeInfoMU.Unlock()
			continue
		}

		p2pOpen := true
		address := fmt.Sprintf("%s:%d", peer.RemoteIP, 26656)
		conn, err := net.DialTimeout("tcp", address, 3*time.Second)
		if err != nil {
			p2pOpen = false
			s.logger.Warningf("port 26656 on node %s unreachable\n", validator.Endpoint)
		}
		if conn != nil {
			defer conn.Close()
		}

		(*nodeInfo)[peerID] = PeerInfo{
			endpoint: endpoint,
			ip:       peer.RemoteIP,
			p2pOpen:  p2pOpen,
			nodeID:   peerID,
		}
		nodeInfoMU.Unlock()
	}
}

// collectAllPeerInfo gathers peer information from all validators in parallel
func (s *Server) collectAllPeerInfo(ctx context.Context, validators []db.CoreValidator) (*map[string]PeerInfo, error) {
	nodeInfo := new(map[string]PeerInfo)
	*nodeInfo = make(map[string]PeerInfo)
	var nodeInfoMU sync.Mutex

	var wg sync.WaitGroup
	wg.Add(len(validators))

	for _, validator := range validators {
		go func(validator db.CoreValidator) {
			defer wg.Done()
			s.gatherPeerInfo(ctx, validator, nodeInfo, &nodeInfoMU)
		}(validator)
	}

	wg.Wait()
	return nodeInfo, nil
}

// selectPeersFromHashring selects peers using a hashring algorithm
func (s *Server) selectPeersFromHashring(validatorIDs []string, amountWantedPeers int, existingPeers []coretypes.Peer) ([]string, error) {
	hr := hashring.New(validatorIDs)
	selectedPeerIDs, ok := hr.GetNodes(s.config.ProposerAddress, amountWantedPeers)
	if !ok {
		return nil, fmt.Errorf("could not get hashring result: %v %v %v", s.config.ProposerAddress, amountWantedPeers, validatorIDs)
	}

	// Remove existing peers
	for _, peer := range existingPeers {
		existingPeerID := string(peer.NodeInfo.DefaultNodeID)
		if slices.Contains(selectedPeerIDs, existingPeerID) {
			idx := slices.Index(selectedPeerIDs, existingPeerID)
			if idx != -1 {
				selectedPeerIDs = slices.Delete(selectedPeerIDs, idx, idx+1)
			}
		}
	}

	return selectedPeerIDs, nil
}

// dialSelectedPeers attempts to connect to the selected peers
func (s *Server) dialSelectedPeers(ctx context.Context, selectedPeerIDs []string, nodeInfo *map[string]PeerInfo) error {
	peers := []string{}
	for _, peerID := range selectedPeerIDs {
		peerInfo, ok := (*nodeInfo)[peerID]
		if !ok {
			continue
		}
		peers = append(peers, fmt.Sprintf("%s@%s:26656", peerID, peerInfo.ip))
	}

	res, err := s.rpc.DialPeers(ctx, peers, false, false, false)
	if err != nil {
		return fmt.Errorf("could not dial peers: %v", err)
	}

	if res.Log != "" {
		s.logger.Infof("dial peers: %s", res.Log)
	}
	return nil
}

// updateCometPeers coordinates the peer update process
func (s *Server) updateCometPeers() error {
	ctx := context.Background()

	// Check if we need more peers
	sufficientPeers, err := s.checkPeerCount(ctx)
	if err != nil {
		return err
	}
	if sufficientPeers {
		return nil
	}

	// Get validator information
	validatorIDToEndpoint, validatorIDs, err := s.getValidatorPeers(ctx)
	if err != nil {
		return err
	}

	// Get current network information
	netInfo, err := s.rpc.NetInfo(ctx)
	if err != nil {
		return err
	}

	// Collect peer information
	validators, err := s.db.GetAllRegisteredNodes(ctx)
	if err != nil {
		return err
	}
	nodeInfo, err := s.collectAllPeerInfo(ctx, validators)
	if err != nil {
		return err
	}

	// Calculate desired number of peers (67% of validators)
	amountWantedPeers := int(math.Ceil(float64(len(validatorIDToEndpoint)) * 0.67))

	// Select peers using hashring
	selectedPeerIDs, err := s.selectPeersFromHashring(validatorIDs, amountWantedPeers, netInfo.Peers)
	if err != nil {
		return err
	}

	// Connect to selected peers
	return s.dialSelectedPeers(ctx, selectedPeerIDs, nodeInfo)
}
