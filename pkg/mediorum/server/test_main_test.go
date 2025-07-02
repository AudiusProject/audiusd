package server

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/AudiusProject/audiusd/pkg/common"
	coreServer "github.com/AudiusProject/audiusd/pkg/core/server"
	"github.com/AudiusProject/audiusd/pkg/pos"
	"github.com/AudiusProject/audiusd/pkg/version"
)

var testNetwork []*MediorumServer

func setupTestNetwork(replicationFactor, serverCount int) []*MediorumServer {

	testBaseDir := "/tmp/mediorum_test"
	os.RemoveAll(testBaseDir)

	network := []Peer{}
	servers := []*MediorumServer{}

	dbUrlTemplate := os.Getenv("dbUrlTemplate")
	if dbUrlTemplate == "" {
		dbUrlTemplate = "postgres://postgres:example@localhost:5454/m%d"
	}

	for i := 1; i <= serverCount; i++ {
		network = append(network, Peer{
			Host:   fmt.Sprintf("http://127.0.0.1:%d", 1980+i),
			Wallet: fmt.Sprintf("0xWallet%d", i), // todo keypair stuff
		})
	}

	for idx, peer := range network {
		peer := peer
		config := MediorumConfig{
			Env:               "test",
			Self:              peer,
			Peers:             network,
			ReplicationFactor: replicationFactor,
			Dir:               fmt.Sprintf("%s/%s", testBaseDir, peer.Wallet),
			PostgresDSN:       fmt.Sprintf(dbUrlTemplate, idx+1),
			VersionJson: version.VersionJson{
				Version: "0.0.0",
				Service: "content-node",
			},
		}
		posChannel := make(chan pos.PoSRequest)
		server, err := New(config, posChannel, &coreServer.CoreService{}, common.NewLogger(&slog.HandlerOptions{}))
		if err != nil {
			panic(err)
		}
		servers = append(servers, server)

		go func() {
			server.MustStart()
		}()
	}

	// give each server time to startup + health check
	time.Sleep(time.Second)
	log.Printf("started %d servers", serverCount)

	return servers

}

func TestMain(m *testing.M) {
	testNetwork = setupTestNetwork(5, 9)

	exitVal := m.Run()
	// todo: tear down testNetwork

	os.Exit(exitVal)
}
