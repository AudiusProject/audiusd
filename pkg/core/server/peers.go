// Peers that core is aware of and uses. This is different than the lower level p2p list that cometbft manages.
// This is where we store sdk clients for other validators for the purposes of forwarding transactions, querying health checks, and
// anything else.
package server

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/AudiusProject/audiusd/pkg/eth/contracts"
	"github.com/AudiusProject/audiusd/pkg/sdk"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/labstack/echo/v4"
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

func (s *Server) startPeerManager(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.awaitRpcReady:
	}

	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			if err := s.onPeerTick(ctx); err != nil {
				s.logger.Errorf("error connecting to peers: %v", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Server) onPeerTick(ctx context.Context) error {
	validators, err := s.db.GetAllRegisteredNodes(ctx)
	if err != nil {
		return fmt.Errorf("could not get validators from db: %v", err)
	}

	netInfo, err := s.rpc.NetInfo(ctx)
	if err != nil {
		return fmt.Errorf("could not get self net info: %v", err)
	}
	for _, peer := range netInfo.Peers {
		s.cometListenAddrs.Set(CometBFTAddress(peer.NodeInfo.ID()), peer.NodeInfo.ListenAddr)
	}

	var wg sync.WaitGroup
	wg.Add(len(validators))

	for _, validator := range validators {
		go s.peerValidator(ctx, &wg, &validator)
	}

	wg.Wait()

	return nil
}

func (s *Server) peerValidator(ctx context.Context, wg *sync.WaitGroup, validator *db.CoreValidator) {
	defer wg.Done()

	endpoint := validator.Endpoint
	ethAddress := validator.EthAddress
	nodeid := strings.ToLower(validator.CometAddress)
	self := s.config.WalletAddress
	logger := s.logger.Child("peer_manager")

	logger.Infof("peering with %s", endpoint)

	// don't peer with self
	if ethAddress == self {
		return
	}

	// get or create connectrpc client
	connectRPC, ok := s.connectRPCPeers.Get(ethAddress)
	if !ok {
		// create connectrpc client
		auds := sdk.NewAudiusdSDK(endpoint)
		connectRPC = auds.Core
		s.connectRPCPeers.Set(ethAddress, connectRPC)
	}

	// get or create cometrpc client
	cometRPC, ok := s.cometRPCPeers.Get(ethAddress)
	if !ok {
		rpc, _ := rpchttp.New(endpoint + "/core/crpc")
		if rpc != nil {
			cometRPC = rpc
			s.cometRPCPeers.Set(ethAddress, cometRPC)
		}
	}

	listener, peered := s.cometListenAddrs.Get(nodeid)
	portAccessible := false
	conn, _ := net.DialTimeout("tcp", listener, 3*time.Second)
	if conn != nil {
		portAccessible = true
		_ = conn.Close()
	}

	if !peered {
		res, err := s.rpc.DialPeers(ctx, []string{listener}, true, true, false)
		if err != nil {
			s.logger.Errorf("error dialing peer %s: %v", endpoint, err)
		} else {
			s.logger.Infof("dialed peer %s: %s", endpoint, res.Log)
			peered = true
		}
	}

	nodeStatus := &v1.GetStatusResponse_PeerInfo_Peer{
		Endpoint:       endpoint,
		CometAddress:   nodeid,
		EthAddress:     ethAddress,
		NodeType:       validator.NodeType,
		Connectrpc:     connectRPC != nil,
		Cometrpc:       cometRPC != nil,
		Cometp2P:       peered,
		PortAccessible: portAccessible,
	}

	upsertCache(s.cache.peers, PeersKey, func(peerInfo *v1.GetStatusResponse_PeerInfo) *v1.GetStatusResponse_PeerInfo {
		var index int = -1
		for i, peer := range peerInfo.Peers {
			if peer.CometAddress == nodeStatus.CometAddress {
				index = i
				break
			}
		}

		if index >= 0 {
			peerInfo.Peers[index] = nodeStatus
		} else {
			peerInfo.Peers = append(peerInfo.Peers, nodeStatus)
		}

		// Sort by CometAddress
		sort.Slice(peerInfo.Peers, func(i, j int) bool {
			return peerInfo.Peers[i].CometAddress < peerInfo.Peers[j].CometAddress
		})

		return peerInfo
	})
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
