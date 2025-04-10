package main

import (
	"log"

	corev1connect "github.com/AudiusProject/audiusd/pkg/api/core/v1/v1connect"
	etlv1connect "github.com/AudiusProject/audiusd/pkg/api/etl/v1/v1connect"
	storagev1connect "github.com/AudiusProject/audiusd/pkg/api/storage/v1/v1connect"
	"github.com/AudiusProject/audiusd/pkg/core"
	"github.com/AudiusProject/audiusd/pkg/etl"
	storage "github.com/AudiusProject/audiusd/pkg/mediorum"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())

	h2s := &http2.Server{}

	coreService := core.NewCoreService(nil)
	corePath, coreHandler := corev1connect.NewCoreServiceHandler(coreService)
	e.Group("").Any(corePath+"*", echo.WrapHandler(h2c.NewHandler(coreHandler, h2s)))

	storageService := storage.NewStorageService()
	storagePath, storageHandler := storagev1connect.NewStorageServiceHandler(storageService)
	e.Group("").Any(storagePath+"*", echo.WrapHandler(h2c.NewHandler(storageHandler, h2s)))

	etlService := etl.NewETLService(nil)
	etlPath, etlHandler := etlv1connect.NewETLServiceHandler(etlService)
	e.Group("").Any(etlPath+"*", echo.WrapHandler(h2c.NewHandler(etlHandler, h2s)))

	if err := e.Start(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
