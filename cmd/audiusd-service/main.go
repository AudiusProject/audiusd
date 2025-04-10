package main

import (
	"log"
	"net/http"

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

	rpcGroup := e.Group("")
	coreService := core.NewCoreService(nil)
	corePath, coreHandler := corev1connect.NewCoreServiceHandler(coreService)
	rpcGroup.Any(corePath+"*", echo.WrapHandler(coreHandler))

	storageService := storage.NewStorageService()
	storagePath, storageHandler := storagev1connect.NewStorageServiceHandler(storageService)
	rpcGroup.Any(storagePath+"*", echo.WrapHandler(storageHandler))

	etlService := etl.NewETLService(nil)
	etlPath, etlHandler := etlv1connect.NewETLServiceHandler(etlService)
	rpcGroup.Any(etlPath+"*", echo.WrapHandler(etlHandler))

	h2s := &http2.Server{}
	h1s := &http.Server{
		Addr:    ":8080",
		Handler: h2c.NewHandler(e, h2s), // 👈 Wrap entire Echo router here
	}

	log.Println("Server listening on http://localhost:8080 (Connect + gRPC over H2C)")
	if err := h1s.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
