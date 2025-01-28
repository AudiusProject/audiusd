package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/contracts"
	"github.com/labstack/echo/v4"
)

type EthNodes struct {
	ethNodes          []*contracts.Node
	duplicateEthNodes []*contracts.Node
	ethNodeMU         sync.RWMutex
}

func (e *EthNodes) setNodes(nodes, duplicateNodes []*contracts.Node) {
	e.ethNodeMU.Lock()
	defer e.ethNodeMU.Unlock()

	e.ethNodes = make([]*contracts.Node, len(nodes))
	copy(e.ethNodes, nodes)

	e.duplicateEthNodes = make([]*contracts.Node, len(duplicateNodes))
	copy(e.duplicateEthNodes, duplicateNodes)
}

func (e *EthNodes) getNodes() ([]*contracts.Node, []*contracts.Node) {
	e.ethNodeMU.RLock()
	defer e.ethNodeMU.RUnlock()

	ethNodes := make([]*contracts.Node, len(e.ethNodes))
	copy(ethNodes, e.ethNodes)

	duplicateEthNodes := make([]*contracts.Node, len(e.duplicateEthNodes))
	copy(duplicateEthNodes, e.duplicateEthNodes)

	return ethNodes, duplicateEthNodes
}

func (s *Server) startEthNodeManager() error {
	// Initial query with retries
	maxRetries := 10
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		if err := s.gatherEthNodes(); err != nil {
			s.logger.Errorf("error gathering registered eth nodes (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(retryDelay)
			retryDelay *= 2
		} else {
			break
		}
		if i == maxRetries-1 {
			return fmt.Errorf("failed to gather registered eth nodes after %d retries", maxRetries)
		}
	}

	close(s.awaitEthNodesReady)
	s.logger.Info("said eth nodes ready")

	ticker := time.NewTicker(6 * time.Hour)
	if s.isDevEnvironment() {
		// query eth chain more aggressively on dev
		ticker = time.NewTicker(5 * time.Second)
	}

	defer ticker.Stop()

	for range ticker.C {
		if err := s.gatherEthNodes(); err != nil {
			s.logger.Errorf("error gathering eth nodes: %v", err)
		}
	}
	return nil
}

func (s *Server) gatherEthNodes() error {
	s.logger.Info("gathering ethereum nodes")

	nodes, err := s.contracts.GetAllRegisteredNodes(context.Background())
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		return fmt.Errorf("got 0 registered nodes: %v", nodes)
	}

	ethNodeMap := make(map[string]*contracts.Node, len(nodes))
	duplicateEthNodeSet := make(map[string]*contracts.Node)

	for _, node := range nodes {
		ethaddr := node.DelegateOwnerWallet.String()
		if existingNode, ok := ethNodeMap[ethaddr]; ok {
			duplicateEthNodeSet[node.Endpoint] = node
			duplicateEthNodeSet[existingNode.Endpoint] = existingNode
		} else {
			ethNodeMap[ethaddr] = node
		}
	}

	duplicateEthNodes := make([]*contracts.Node, 0, len(duplicateEthNodeSet))
	for _, node := range duplicateEthNodeSet {
		duplicateEthNodes = append(duplicateEthNodes, node)
	}

	s.ethNodes.setNodes(nodes, duplicateEthNodes)

	return nil
}

func (s *Server) blacklistDuplicateEthNodes() error {
	return nil
}

func (s *Server) getEthNodesHandler(c echo.Context) error {
	ethNodes, duplicateEthNodes := s.ethNodes.getNodes()
	res := struct {
		Nodes          []*contracts.Node `json:"nodes"`
		DuplicateNodes []*contracts.Node `json:"duplicateNodes"`
	}{
		Nodes:          ethNodes,
		DuplicateNodes: duplicateEthNodes,
	}
	return c.JSON(200, res)
}
