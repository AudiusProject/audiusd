package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/davecgh/go-spew/spew"
)

func TestPeers(t *testing.T) {
	nodesEndpoint := "https://discoveryprovider.audius.co/core/nodes"
	resp, err := http.Get(nodesEndpoint)
	if err != nil {
		t.Errorf("Failed to get nodes from endpoint %s", nodesEndpoint)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Data []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Errorf("Failed to decode response body: %v", err)
		return
	}

	endpoints := result.Data

	s := Server{
		logger: common.NewLogger(nil),
	}

	cometPeers, err := s.collectCometPeers(endpoints)
	if err != nil {
		t.Errorf("Failed to collect comet peers: %v", err)
		return
	}

	spew.Dump(cometPeers)
}
