package main

import (
	"log"

	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/console"
	"github.com/AudiusProject/audiusd/pkg/etl"
	"github.com/AudiusProject/audiusd/pkg/sdk"
)

func main() {
	logger := common.NewLogger(nil)

	logger.Info("Starting Console")

	auds := sdk.NewAudiusdSDK("http://localhost:3000")

	etl := etl.NewETLService(auds.Core, logger)
	etl.SetDBURL("postgres://postgres:postgres@0.0.0.0:5432/audiusd?sslmode=disable")
	etl.SetCheckReadiness(false)

	console := console.NewConsole(etl)
	console.SetupRoutes()

	defer console.Stop()
	if err := console.Run(); err != nil {
		log.Fatal(err)
	}
}
