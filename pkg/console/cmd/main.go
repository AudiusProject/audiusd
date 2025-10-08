package main

import (
	"log"

	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/console"
	"github.com/AudiusProject/audiusd/pkg/etl"
	"github.com/AudiusProject/audiusd/pkg/sdk"
	"go.uber.org/zap"
)

func main() {
	logger := common.NewLogger(nil)

	logger.Info("Starting Console")

	// Connect to postgres from Makefile (port 5444)
	dbURL := "postgres://postgres:postgres@0.0.0.0:5444/audiusd?sslmode=disable"

	logger.Info("pgURL", "url", dbURL)

	auds := sdk.NewAudiusdSDK("creatornode.audius.co")

	etl := etl.NewETLService(auds.Core, zap.NewNop())
	etl.SetDBURL(dbURL)
	etl.SetCheckReadiness(false)

	console := console.NewConsole(etl, nil, "prod")
	console.Initialize()

	defer console.Stop()
	if err := console.Run(); err != nil {
		log.Fatal(err)
	}
}
