// Peers that core is aware of and uses. This is different than the lower level p2p list that cometbft manages.
// This is where we store sdk clients for other validators for the purposes of forwarding transactions, querying health checks, and
// anything else.
package server

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/contracts"
	"github.com/AudiusProject/audiusd/pkg/core/sdk"
	"github.com/labstack/echo/v4"
)

type Peers struct {
	peers map[string]*sdk.Sdk
	peersMU sync.RWMutex
}

// UpdatePeers updates the peers map
func (p *Peers) UpdatePeers(newPeers map[string]*sdk.Sdk) {
	p.peersMU.Lock()
	defer p.peersMU.Unlock()
	p.peers = newPeers
}

// GetPeers retrieves a snapshot of the current peers map
func (p *Peers) GetPeers() map[string]*sdk.Sdk {
	p.peersMU.RLock()
	defer p.peersMU.RUnlock()
	// Return a copy to avoid race conditions
	peersCopy := make(map[string]*sdk.Sdk, len(p.peers))
	for k, v := range p.peers {
		peersCopy[k] = v
	}
	return peersCopy
}

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

func (s *Server) startPeerManager() error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.onPeerTick(); err != nil {
			s.logger.Errorf("error connecting to peers: %v", err)
		}
	}
	return nil
}

func (s *Server) onPeerTick() error {
	validators, err := s.db.GetAllRegisteredNodes(context.Background())
	if err != nil {
		return fmt.Errorf("could not get validators from db: %v", err)
	}

	peers := s.peers.GetPeers()
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
				s.logger.Errorf("could not peer with '%s', error: %v", oapiendpoint, err)
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
		s.peers.UpdatePeers(peers)
	}

	return nil
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
