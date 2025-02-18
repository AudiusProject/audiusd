package server

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "embed"
	_ "net/http/pprof"

	core "github.com/AudiusProject/audiusd/pkg/core/sdk"
	"github.com/AudiusProject/audiusd/pkg/mediorum/cidutil"
	"github.com/AudiusProject/audiusd/pkg/mediorum/crudr"
	"github.com/AudiusProject/audiusd/pkg/mediorum/ethcontracts"
	"github.com/AudiusProject/audiusd/pkg/mediorum/persistence"
	"github.com/AudiusProject/audiusd/pkg/pos"
	"github.com/erni27/imcache"
	"github.com/imroc/req/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/oschwald/maxminddb-golang"
	"gocloud.dev/blob"
	"golang.org/x/exp/slog"

	_ "gocloud.dev/blob/fileblob"
)

type Peer struct {
	Host   string `json:"host"`
	Wallet string `json:"wallet"`
}

type VersionJson struct {
	Version string `json:"version"`
	Service string `json:"service"`
}

type MediorumConfig struct {
	Env                       string
	Self                      Peer
	Peers                     []Peer
	Signers                   []Peer
	ReplicationFactor         int
	Dir                       string `default:"/tmp/mediorum"`
	BlobStoreDSN              string `json:"-"`
	MoveFromBlobStoreDSN      string `json:"-"`
	PostgresDSN               string `json:"-"`
	PrivateKey                string `json:"-"`
	ListenPort                string
	TrustedNotifierID         int
	SPID                      int
	SPOwnerWallet             string
	GitSHA                    string
	AudiusDockerCompose       string
	AutoUpgradeEnabled        bool
	WalletIsRegistered        bool
	StoreAll                  bool
	VersionJson               VersionJson
	DiscoveryListensEndpoints []string
	CoreGRPCEndpoint          string

	// should have a basedir type of thing
	// by default will put db + blobs there

	privateKey *ecdsa.PrivateKey
}

type MediorumServer struct {
	echo             *echo.Echo
	bucket           *blob.Bucket
	logger           *slog.Logger
	crud             *crudr.Crudr
	pgPool           *pgxpool.Pool
	quit             chan os.Signal
	trustedNotifier  *ethcontracts.NotifierInfo
	reqClient        *req.Client
	rendezvousHasher *RendezvousHasher
	transcodeWork    chan *Upload

	// stats
	statsMutex       sync.RWMutex
	transcodeStats   *TranscodeStats
	mediorumPathUsed uint64
	mediorumPathSize uint64
	mediorumPathFree uint64

	databaseSize          uint64
	dbSizeErr             string
	lastSuccessfulRepair  RepairTracker
	lastSuccessfulCleanup RepairTracker

	uploadsCount    int64
	uploadsCountErr string

	isSeeding        bool
	isAudiusdManaged bool

	peerHealthsMutex      sync.RWMutex
	peerHealths           map[string]*PeerHealth
	unreachablePeers      []string
	redirectCache         *imcache.Cache[string, string]
	uploadOrigCidCache    *imcache.Cache[string, string]
	imageCache            *imcache.Cache[string, []byte]
	failsPeerReachability bool

	StartedAt time.Time
	Config    MediorumConfig

	crudSweepMutex sync.Mutex

	// handle communication between core and mediorum for Proof of Storage
	posChannel chan pos.PoSRequest

	coreSdk      *core.Sdk
	coreSdkReady chan struct{}

	geoIPdb      *maxminddb.Reader
	geoIPdbReady chan struct{}

	playEventQueue *PlayEventQueue
}

type PeerHealth struct {
	Version        string               `json:"version"`
	LastReachable  time.Time            `json:"lastReachable"`
	LastHealthy    time.Time            `json:"lastHealthy"`
	ReachablePeers map[string]time.Time `json:"reachablePeers"`
}

var (
	apiBasePath = ""
)

const PercentSeededThreshold = 50

func New(config MediorumConfig, posChannel chan pos.PoSRequest) (*MediorumServer, error) {
	if env := os.Getenv("MEDIORUM_ENV"); env != "" {
		config.Env = env
	}

	var isAudiusdManaged bool
	if audiusdGenerated := os.Getenv("AUDIUS_D_GENERATED"); audiusdGenerated != "" {
		isAudiusdManaged = true
	}

	if config.VersionJson == (VersionJson{}) {
		log.Fatal(".version.json is required to be bundled with the mediorum binary")
	}

	// validate host config
	if config.Self.Host == "" {
		log.Fatal("host is required")
	} else if hostUrl, err := url.Parse(config.Self.Host); err != nil {
		log.Fatal("invalid host: ", err)
	} else if config.ListenPort == "" {
		config.ListenPort = hostUrl.Port()
	}

	if config.Dir == "" {
		config.Dir = "/tmp/mediorum"
	}

	if config.BlobStoreDSN == "" {
		config.BlobStoreDSN = "file://" + config.Dir + "/blobs?no_tmp_dir=true"
	}

	if pk, err := ethcontracts.ParsePrivateKeyHex(config.PrivateKey); err != nil {
		log.Println("invalid private key: ", err)
	} else {
		config.privateKey = pk
	}

	// check that we're registered...
	for _, peer := range config.Peers {
		if strings.EqualFold(config.Self.Wallet, peer.Wallet) && strings.EqualFold(config.Self.Host, peer.Host) {
			config.WalletIsRegistered = true
			break
		}
	}

	logger := slog.With("self", config.Self.Host)

	if config.discoveryListensEnabled() {
		logger.Info("discovery listens enabled")
	}

	// ensure dir
	if err := os.MkdirAll(config.Dir, os.ModePerm); err != nil {
		logger.Error("failed to create local persistent storage dir", "err", err)
	}

	// bucket
	bucket, err := persistence.Open(config.BlobStoreDSN)
	if err != nil {
		logger.Error("failed to open persistent storage bucket", "err", err)
		return nil, err
	}

	// bucket to move all files from
	if config.MoveFromBlobStoreDSN != "" {
		if config.MoveFromBlobStoreDSN == config.BlobStoreDSN {
			logger.Error("AUDIUS_STORAGE_DRIVER_URL_MOVE_FROM cannot be the same as AUDIUS_STORAGE_DRIVER_URL")
			return nil, err
		}
		bucketToMoveFrom, err := persistence.Open(config.MoveFromBlobStoreDSN)
		if err != nil {
			logger.Error("Failed to open bucket to move from. Ensure AUDIUS_STORAGE_DRIVER_URL and AUDIUS_STORAGE_DRIVER_URL_MOVE_FROM are set (the latter can be empty if not moving data)", "err", err)
			return nil, err
		}

		logger.Info(fmt.Sprintf("Moving all files from %s to %s. This may take a few hours...", config.MoveFromBlobStoreDSN, config.BlobStoreDSN))
		err = persistence.MoveAllFiles(bucketToMoveFrom, bucket)
		if err != nil {
			logger.Error("Failed to move files. Ensure AUDIUS_STORAGE_DRIVER_URL and AUDIUS_STORAGE_DRIVER_URL_MOVE_FROM are set (the latter can be empty if not moving data)", "err", err)
			return nil, err
		}

		logger.Info("Finished moving files between buckets. Please remove AUDIUS_STORAGE_DRIVER_URL_MOVE_FROM from your environment and restart the server.")
	}

	// db
	db := dbMustDial(config.PostgresDSN)
	if config.Env == "dev" {
		// air doesn't reset client connections so this explicitly sets the client encoding
		sqlDB, err := db.DB()
		if err == nil {
			_, err = sqlDB.Exec("SET client_encoding TO 'UTF8';")
			if err != nil {
				panic(fmt.Sprintf("Failed to set client encoding: %v", err))
			}
		}
	}

	// pg pool
	// config.PostgresDSN
	pgConfig, _ := pgxpool.ParseConfig(config.PostgresDSN)
	pgPool, err := pgxpool.NewWithConfig(context.Background(), pgConfig)
	if err != nil {
		logger.Error("dial postgres failed", "err", err)
	}

	// crud
	peerHosts := []string{}
	allHosts := []string{}
	for _, peer := range config.Peers {
		allHosts = append(allHosts, peer.Host)
		if peer.Host != config.Self.Host {
			peerHosts = append(peerHosts, peer.Host)
		}
	}
	crud := crudr.New(config.Self.Host, config.privateKey, peerHosts, db)
	dbMigrate(crud, config.Self.Host)

	rendezvousHasher := NewRendezvousHasher(allHosts)

	// req.cool http client
	reqClient := req.C().
		SetUserAgent("mediorum " + config.Self.Host).
		SetTimeout(5 * time.Second)

	// Read trusted notifier endpoint from chain
	var trustedNotifier ethcontracts.NotifierInfo
	if config.TrustedNotifierID > 0 {
		trustedNotifier, err = ethcontracts.GetNotifierForID(strconv.Itoa(config.TrustedNotifierID), config.Self.Wallet)
		if err == nil {
			logger.Info("got trusted notifier from chain", "endpoint", trustedNotifier.Endpoint, "wallet", trustedNotifier.Wallet)
		} else {
			logger.Error("failed to get trusted notifier from chain, not polling delist statuses", "err", err)
		}
	} else {
		logger.Warn("trusted notifier id not set, not polling delist statuses or serving /contact route")
	}

	// echoServer server
	echoServer := echo.New()
	echoServer.HideBanner = true
	echoServer.Debug = true

	echoServer.Use(middleware.Recover())
	echoServer.Use(middleware.Logger())
	echoServer.Use(middleware.CORS())
	echoServer.Use(timingMiddleware)

	ss := &MediorumServer{
		echo:             echoServer,
		bucket:           bucket,
		crud:             crud,
		pgPool:           pgPool,
		reqClient:        reqClient,
		logger:           logger,
		quit:             make(chan os.Signal, 1),
		trustedNotifier:  &trustedNotifier,
		isSeeding:        config.Env == "stage" || config.Env == "prod",
		isAudiusdManaged: isAudiusdManaged,
		rendezvousHasher: rendezvousHasher,
		transcodeWork:    make(chan *Upload),
		posChannel:       posChannel,

		peerHealths:        map[string]*PeerHealth{},
		redirectCache:      imcache.New(imcache.WithMaxEntriesLimitOption[string, string](50_000, imcache.EvictionPolicyLRU)),
		uploadOrigCidCache: imcache.New(imcache.WithMaxEntriesLimitOption[string, string](50_000, imcache.EvictionPolicyLRU)),
		imageCache:         imcache.New(imcache.WithMaxEntriesLimitOption[string, []byte](10_000, imcache.EvictionPolicyLRU)),

		StartedAt:    time.Now().UTC(),
		Config:       config,
		coreSdkReady: make(chan struct{}),
		geoIPdbReady: make(chan struct{}),

		playEventQueue: NewPlayEventQueue(),
	}

	routes := echoServer.Group(apiBasePath)

	routes.GET("", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/dashboard/#/nodes/content-node?endpoint="+config.Self.Host)
	})
	routes.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/dashboard/#/nodes/content-node?endpoint="+config.Self.Host)
	})

	// public: uploads
	routes.GET("/uploads", ss.serveUploadList)
	routes.GET("/uploads/:id", ss.serveUploadDetail, ss.requireHealthy)
	routes.POST("/uploads/:id", ss.updateUpload, ss.requireHealthy, ss.requireUserSignature)
	routes.POST("/uploads", ss.postUpload, ss.requireHealthy)
	// workaround because reverse proxy catches the browser's preflight OPTIONS request instead of letting our CORS middleware handle it
	routes.OPTIONS("/uploads", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	routes.POST("/generate_preview/:cid/:previewStartSeconds", ss.generatePreview, ss.requireHealthy)

	// legacy blob audio analysis
	routes.GET("/tracks/legacy/:cid/analysis", ss.serveLegacyBlobAnalysis, ss.requireHealthy)

	// serve blob (audio)
	routes.HEAD("/ipfs/:cid", ss.serveBlob, ss.requireHealthy, ss.ensureNotDelisted)
	routes.GET("/ipfs/:cid", ss.serveBlob, ss.requireHealthy, ss.ensureNotDelisted)
	routes.HEAD("/content/:cid", ss.serveBlob, ss.requireHealthy, ss.ensureNotDelisted)
	routes.GET("/content/:cid", ss.serveBlob, ss.requireHealthy, ss.ensureNotDelisted)
	routes.HEAD("/tracks/cidstream/:cid", ss.serveBlob, ss.requireHealthy, ss.ensureNotDelisted, ss.requireRegisteredSignature)
	routes.GET("/tracks/cidstream/:cid", ss.serveBlob, ss.requireHealthy, ss.ensureNotDelisted, ss.requireRegisteredSignature)

	// serve image
	routes.HEAD("/ipfs/:jobID/:variant", ss.serveImage, ss.requireHealthy)
	routes.GET("/ipfs/:jobID/:variant", ss.serveImage, ss.requireHealthy)
	routes.HEAD("/content/:jobID/:variant", ss.serveImage, ss.requireHealthy)
	routes.GET("/content/:jobID/:variant", ss.serveImage, ss.requireHealthy)

	routes.GET("/contact", ss.serveContact)
	routes.GET("/health_check", ss.serveHealthCheck)
	routes.HEAD("/health_check", ss.serveHealthCheck)
	routes.GET("/ip_check", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"data": c.RealIP(), // client/requestor IP
		})
	})

	routes.GET("/delist_status/track/:trackCid", ss.serveTrackDelistStatus)
	routes.GET("/delist_status/user/:userId", ss.serveUserDelistStatus)
	routes.POST("/delist_status/insert", ss.serveInsertDelistStatus, ss.requireBodySignedByOwner)

	// -------------------
	// healthz
	healthz := routes.Group("/healthz")
	healthzUrl, err := url.Parse("http://healthz")
	if err != nil {
		log.Fatal("Invalid healthz URL: ", err)
	}
	healthzProxy := httputil.NewSingleHostReverseProxy(healthzUrl)
	healthz.Any("*", echo.WrapHandler(healthzProxy))

	// -------------------
	// reverse proxy /d and /d_api to uptime container
	uptimeUrl, err := url.Parse("http://uptime:1996")
	if err != nil {
		log.Fatal("Invalid uptime URL: ", err)
	}
	uptimeProxy := httputil.NewSingleHostReverseProxy(uptimeUrl)

	uptimeAPI := routes.Group("/d_api")
	// fixes what I think should be considered an echo bug: https://github.com/labstack/echo/issues/1419
	uptimeAPI.Use(ACAOHeaderOverwriteMiddleware)
	uptimeAPI.Any("/*", echo.WrapHandler(uptimeProxy))

	uptimeUI := routes.Group("/d")
	uptimeUI.Any("*", echo.WrapHandler(uptimeProxy))

	// -------------------
	// internal
	internalApi := routes.Group("/internal")

	// internal: crud
	internalApi.GET("/crud/sweep", ss.serveCrudSweep)
	internalApi.POST("/crud/push", ss.serveCrudPush, middleware.BasicAuth(ss.checkBasicAuth))

	internalApi.GET("/blobs/location/:cid", ss.serveBlobLocation, cidutil.UnescapeCidParam)
	internalApi.GET("/blobs/info/:cid", ss.serveBlobInfo, cidutil.UnescapeCidParam)

	// internal: blobs between peers
	internalApi.GET("/blobs/:cid", ss.serveInternalBlobGET, cidutil.UnescapeCidParam, middleware.BasicAuth(ss.checkBasicAuth))
	internalApi.POST("/blobs", ss.serveInternalBlobPOST, middleware.BasicAuth(ss.checkBasicAuth))
	internalApi.GET("/qm.csv", ss.serveInternalQmCsv)

	// WIP internal: metrics
	internalApi.GET("/metrics", ss.getMetrics)
	internalApi.GET("/metrics/blobs-served/:timeRange", ss.getBlobsServedMetrics)
	internalApi.GET("/logs/partition-ops", ss.getPartitionOpsLog)
	internalApi.GET("/logs/reaper", ss.getReaperLog)
	internalApi.GET("/logs/repair", ss.serveRepairLog)
	internalApi.GET("/logs/storageAndDb", ss.serveStorageAndDbLogs)
	internalApi.GET("/logs/pg-upgrade", ss.getPgUpgradeLog)

	// internal: testing
	internalApi.GET("/proxy_health_check", ss.proxyHealthCheck)

	go ss.loadGeoIPDatabase()
	go ss.initCoreSdk()

	return ss, nil

}

func setResponseACAOHeaderFromRequest(req http.Request, resp echo.Response) {
	resp.Header().Set(
		echo.HeaderAccessControlAllowOrigin,
		req.Header.Get(echo.HeaderOrigin),
	)
}

func ACAOHeaderOverwriteMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		ctx.Response().Before(func() {
			setResponseACAOHeaderFromRequest(*ctx.Request(), *ctx.Response())
		})
		return next(ctx)
	}
}

func timingMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		startTime := time.Now()
		c.Set("startTime", startTime)
		c.Response().Before(func() {
			c.Response().Header().Set("x-took", time.Since(startTime).String())
		})
		return next(c)
	}
}

// Calling echo response functions (c.JSON or c.String)
// will automatically set timing header in timingMiddleware.
// But for places where we do http.ServeContent
// we have to manually call setTimingHeader right before writing response.
func setTimingHeader(c echo.Context) {
	if startTime, ok := c.Get("startTime").(time.Time); ok {
		c.Response().Header().Set("x-took", time.Since(startTime).String())
	}
}

func (ss *MediorumServer) MustStart() {

	// start pprof server
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	// start server
	go func() {
		err := ss.echo.Start(":" + ss.Config.ListenPort)
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	go ss.startTranscoder()
	go ss.startAudioAnalyzer()

	if ss.Config.StoreAll {
		go ss.startFixTruncatedQmWorker()
	}

	zeroTime := time.Time{}
	var lastSuccessfulRepair RepairTracker
	err := ss.crud.DB.
		Where("finished_at is not null and finished_at != ? and aborted_reason = ?", zeroTime, "").
		Order("started_at desc").
		First(&lastSuccessfulRepair).Error
	if err != nil {
		lastSuccessfulRepair = RepairTracker{Counters: map[string]int{}}
	}
	ss.lastSuccessfulRepair = lastSuccessfulRepair

	var lastSuccessfulCleanup RepairTracker
	err = ss.crud.DB.
		Where("finished_at is not null and finished_at != ? and aborted_reason = ? and cleanup_mode = true", zeroTime, "").
		Order("started_at desc").
		First(&lastSuccessfulCleanup).Error
	if err != nil {
		lastSuccessfulCleanup = RepairTracker{Counters: map[string]int{}}
	}
	ss.lastSuccessfulCleanup = lastSuccessfulCleanup

	// for any background task that make authenticated peer requests
	// only start if we have a valid registered wallet
	if ss.Config.WalletIsRegistered {

		go ss.startHealthPoller()

		go ss.startRepairer()

		go ss.startQmSyncer()

		ss.crud.StartClients()

		go ss.startPollingDelistStatuses()

		go ss.pollForSeedingCompletion()

		go ss.startUploadScroller()

		go ss.startPlayEventQueue()

	} else {
		go func() {
			for range time.Tick(10 * time.Second) {
				ss.logger.Warn("node not fully running yet - please register at https://dashboard.audius.org and restart the server")
			}
		}()
	}

	go ss.monitorMetrics()

	go ss.monitorPeerReachability()

	go ss.startPoSHandler()

	// signals
	signal.Notify(ss.quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-ss.quit
	close(ss.quit)

	ss.Stop()
}

func (ss *MediorumServer) Stop() {
	ss.logger.Info("stopping")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := ss.echo.Shutdown(ctx); err != nil {
		ss.logger.Error("echo shutdown", "err", err)
	}

	if db, err := ss.crud.DB.DB(); err == nil {
		if err := db.Close(); err != nil {
			ss.logger.Error("db shutdown", "err", err)
		}
	}

	// todo: stop transcode worker + repairer too

	ss.logger.Info("bye")

}

func (ss *MediorumServer) pollForSeedingCompletion() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		if ss.crud.GetPercentNodesSeeded() > PercentSeededThreshold {
			ss.isSeeding = false
			return
		}
	}
}

// discovery listens are enabled if endpoints are provided
func (mc *MediorumConfig) discoveryListensEnabled() bool {
	return len(mc.DiscoveryListensEndpoints) > 0
}
