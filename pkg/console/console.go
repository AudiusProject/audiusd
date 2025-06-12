package console

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/etl/v1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/console/templates/pages"
	"github.com/AudiusProject/audiusd/pkg/etl"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"

	"embed"
)

//go:embed assets/css
var cssFS embed.FS

//go:embed assets/images
var imagesFS embed.FS

//go:embed assets/js
var jsFS embed.FS

type Console struct {
	e      *echo.Echo
	etl    *etl.ETLService
	logger *common.Logger
}

func NewConsole(etl *etl.ETLService) *Console {
	return &Console{etl: etl, e: echo.New(), logger: common.NewLogger(nil).Child("console")}
}

func (con *Console) SetupRoutes() {
	e := con.e
	e.HideBanner = true

	// Add cache control middleware for static assets
	cacheControl := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			// Only apply caching to image files
			if strings.HasPrefix(path, "/assets/") && (strings.HasSuffix(path, ".svg") || strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".gif")) {
				c.Response().Header().Set("Cache-Control", "public, max-age=604800") // Cache for 1 week
			}
			return next(c)
		}
	}

	cssHandler := echo.MustSubFS(cssFS, "assets/css")
	imagesHandler := echo.MustSubFS(imagesFS, "assets/images")
	jsHandler := echo.MustSubFS(jsFS, "assets/js")
	e.StaticFS("/assets/css", cssHandler)
	e.StaticFS("/assets/images", imagesHandler)
	e.StaticFS("/assets/js", jsHandler)

	// Apply cache control middleware to static asset routes
	e.Use(cacheControl)

	e.GET("/", con.Dashboard)
	e.GET("/hello", con.Hello)

	// SSE endpoints
	e.GET("/sse/plays", con.LivePlaysSSE)

	// HTMX Fragment routes
	e.GET("/fragments/stats-header", con.StatsHeaderFragment)
	e.GET("/fragments/network-sidebar", con.NetworkSidebarFragment)
	e.GET("/fragments/tps", con.TPSFragment)
	e.GET("/fragments/total-transactions", con.TotalTransactionsFragment)

	e.GET("/validators", con.Validators)
	e.GET("/validator/:address", con.Validator)

	e.GET("/blocks", con.Blocks)
	e.GET("/block/:height", con.Block)

	e.GET("/transactions", con.Transactions)
	e.GET("/transaction/:hash", con.Transaction)

	e.GET("/account/:address", con.stubRoute)
	e.GET("/account/:address/transactions", con.stubRoute)
	e.GET("/account/:address/uploads", con.stubRoute)
	e.GET("/account/:address/releases", con.stubRoute)

	e.GET("/content", con.Content)
	e.GET("/content/:address", con.Content)

	e.GET("/release/:address", con.stubRoute)

	e.GET("/search", con.stubRoute)
}

func (con *Console) Run() error {
	g, _ := errgroup.WithContext(context.Background())

	g.Go(func() error {
		if err := con.etl.Run(); err != nil {
			return err
		}
		return nil
	})

	g.Go(func() error {
		if err := con.e.Start(":8080"); err != nil {
			return err
		}
		return nil
	})

	return g.Wait()
}

func (con *Console) Stop() {
	con.e.Shutdown(context.Background())
}

func (con *Console) Hello(c echo.Context) error {
	param := "sup"
	if name := c.QueryParam("name"); name != "" {
		param = name
	}
	p := pages.Hello(param)
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) Dashboard(c echo.Context) error {
	// Get recent blocks for the dashboard
	blocks, err := con.etl.GetBlocks(c.Request().Context(), &connect.Request[v1.GetBlocksRequest]{
		Msg: &v1.GetBlocksRequest{
			Limit:  10,
			Offset: 0,
		},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get blocks")
	}

	// Get some recent transactions for the dashboard
	transactions, blockHeights, err := con.etl.GetTransactionsForAPI(c.Request().Context(), 10, 0)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get transactions")
	}

	// Get dashboard stats from ETL service
	statsResp, err := con.etl.GetStats(c.Request().Context(), &connect.Request[v1.GetStatsRequest]{
		Msg: &v1.GetStatsRequest{},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get dashboard stats")
	}

	stats := &pages.DashboardStats{
		CurrentBlockHeight:  statsResp.Msg.CurrentBlockHeight,
		ChainID:             statsResp.Msg.ChainId,
		BPS:                 statsResp.Msg.Bps,
		TPS:                 statsResp.Msg.Tps,
		TotalTransactions:   statsResp.Msg.TotalTransactions,
		ValidatorCount:      statsResp.Msg.ValidatorCount,
		LatestBlock:         statsResp.Msg.LatestBlock,
		RecentProposers:     statsResp.Msg.RecentProposers,
		IsSyncing:           statsResp.Msg.SyncStatus != nil && statsResp.Msg.SyncStatus.IsSyncing,
		LatestIndexedHeight: statsResp.Msg.SyncStatus.GetLatestIndexedHeight(),
		LatestChainHeight:   statsResp.Msg.SyncStatus.GetLatestChainHeight(),
		BlockDelta:          statsResp.Msg.SyncStatus.GetBlockDelta(),
	}

	// Convert transaction breakdown from RPC response
	transactionBreakdown := make([]*pages.TransactionTypeBreakdown, len(statsResp.Msg.TransactionBreakdown))
	colors := []string{"bg-blue-500", "bg-green-500", "bg-yellow-500", "bg-purple-500", "bg-red-500", "bg-indigo-500", "bg-pink-500", "bg-gray-500"}
	for i, breakdown := range statsResp.Msg.TransactionBreakdown {
		color := colors[i%len(colors)] // Cycle through colors
		transactionBreakdown[i] = &pages.TransactionTypeBreakdown{
			Type:  breakdown.Type,
			Count: breakdown.Count,
			Color: color,
		}
	}

	p := pages.Dashboard(stats, transactionBreakdown, blocks.Msg.Blocks, transactions.Transactions, blockHeights)
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) Validators(c echo.Context) error {
	// Parse query parameters
	pageParam := c.QueryParam("page")
	countParam := c.QueryParam("count")
	queryType := c.QueryParam("type") // "active", "registrations", "deregistrations"

	page := int32(1) // default to page 1
	if pageParam != "" {
		if parsedPage, err := strconv.ParseInt(pageParam, 10, 32); err == nil && parsedPage > 0 {
			page = int32(parsedPage)
		}
	}

	count := int32(50) // default to 50 per page
	if countParam != "" {
		if parsedCount, err := strconv.ParseInt(countParam, 10, 32); err == nil && parsedCount > 0 && parsedCount <= 200 {
			count = int32(parsedCount)
		}
	}

	// Calculate offset from page number
	offset := (page - 1) * count

	// Default to active validators
	if queryType == "" {
		queryType = "active"
	}

	// Build request based on query type
	var validatorsReq *v1.GetValidatorsRequest
	switch queryType {
	case "active":
		validatorsReq = &v1.GetValidatorsRequest{
			Query:  &v1.GetValidatorsRequest_GetRegisteredValidators{GetRegisteredValidators: &v1.GetRegisteredValidators{}},
			Limit:  count,
			Offset: offset,
		}
	case "registrations":
		validatorsReq = &v1.GetValidatorsRequest{
			Query:  &v1.GetValidatorsRequest_GetValidatorRegistrations{GetValidatorRegistrations: &v1.GetValidatorRegistrations{}},
			Limit:  count,
			Offset: offset,
		}
	case "deregistrations":
		validatorsReq = &v1.GetValidatorsRequest{
			Query:  &v1.GetValidatorsRequest_GetValidatorDeregistrations{GetValidatorDeregistrations: &v1.GetValidatorDeregistrations{}},
			Limit:  count,
			Offset: offset,
		}
	default:
		queryType = "active"
		validatorsReq = &v1.GetValidatorsRequest{
			Query:  &v1.GetValidatorsRequest_GetRegisteredValidators{GetRegisteredValidators: &v1.GetRegisteredValidators{}},
			Limit:  count,
			Offset: offset,
		}
	}

	validators, err := con.etl.GetValidators(c.Request().Context(), &connect.Request[v1.GetValidatorsRequest]{
		Msg: validatorsReq,
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get validators")
	}

	// Calculate pagination state
	hasNext := validators.Msg.HasMore
	hasPrev := page > 1

	p := pages.Validators(validators.Msg.Validators, page, hasNext, hasPrev, count, queryType)
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) Validator(c echo.Context) error {
	address := c.Param("address")
	if address == "" {
		return c.String(http.StatusBadRequest, "Validator address required")
	}

	validator, err := con.etl.GetValidator(c.Request().Context(), &connect.Request[v1.GetValidatorRequest]{
		Msg: &v1.GetValidatorRequest{
			Identifier: &v1.GetValidatorRequest_Address{Address: address},
		},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get validator")
	}

	p := pages.Validator(validator.Msg.Validator, validator.Msg.Events)
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) Blocks(c echo.Context) error {
	// Parse query parameters
	pageParam := c.QueryParam("page")
	countParam := c.QueryParam("count")

	page := int32(1) // default to page 1
	if pageParam != "" {
		if parsedPage, err := strconv.ParseInt(pageParam, 10, 32); err == nil && parsedPage > 0 {
			page = int32(parsedPage)
		}
	}

	count := int32(50) // default to 50 per page
	if countParam != "" {
		if parsedCount, err := strconv.ParseInt(countParam, 10, 32); err == nil && parsedCount > 0 && parsedCount <= 200 {
			count = int32(parsedCount)
		}
	}

	// Calculate offset from page number
	offset := (page - 1) * count

	blocks, err := con.etl.GetBlocks(c.Request().Context(), &connect.Request[v1.GetBlocksRequest]{
		Msg: &v1.GetBlocksRequest{
			Limit:  count,
			Offset: offset,
		},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get blocks")
	}

	// Calculate pagination state
	hasNext := blocks.Msg.HasMore
	hasPrev := page > 1

	p := pages.Blocks(blocks.Msg.Blocks, page, hasNext, hasPrev, count)
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) Transactions(c echo.Context) error {
	// Parse query parameters
	pageParam := c.QueryParam("page")
	countParam := c.QueryParam("count")

	page := int32(1) // default to page 1
	if pageParam != "" {
		if parsedPage, err := strconv.ParseInt(pageParam, 10, 32); err == nil && parsedPage > 0 {
			page = int32(parsedPage)
		}
	}

	count := int32(50) // default to 50 per page
	if countParam != "" {
		if parsedCount, err := strconv.ParseInt(countParam, 10, 32); err == nil && parsedCount > 0 && parsedCount <= 200 {
			count = int32(parsedCount)
		}
	}

	// Calculate offset from page number
	offset := (page - 1) * count

	transactions, blockHeights, err := con.etl.GetTransactionsForAPI(c.Request().Context(), count, offset)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get transactions")
	}

	// Calculate pagination state
	hasNext := transactions.HasMore
	hasPrev := page > 1

	p := pages.Transactions(transactions.Transactions, blockHeights, page, hasNext, hasPrev, count)
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) Content(c echo.Context) error {
	p := pages.Content()
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) Block(c echo.Context) error {
	height, err := strconv.ParseInt(c.Param("height"), 10, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid block height")
	}
	block, err := con.etl.GetBlock(c.Request().Context(), &connect.Request[v1.GetBlockRequest]{
		Msg: &v1.GetBlockRequest{
			Height: int64(height),
		},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get block")
	}
	p := pages.Block(block.Msg.Block)
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) Transaction(c echo.Context) error {
	txHash := c.Param("hash")
	if txHash == "" {
		return c.String(http.StatusBadRequest, "Transaction hash required")
	}

	// Get transaction details using the standard gRPC call
	response, err := con.etl.GetTransaction(c.Request().Context(), &connect.Request[v1.GetTransactionRequest]{
		Msg: &v1.GetTransactionRequest{
			TxHash: txHash,
		},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get transaction")
	}

	p := pages.Transaction(response.Msg.Transaction)
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) stubRoute(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func (con *Console) APIBlocks(c echo.Context) error {
	// Parse query parameters
	limitParam := c.QueryParam("limit")
	offsetParam := c.QueryParam("offset")

	limit := int32(50) // default
	if limitParam != "" {
		if parsedLimit, err := strconv.ParseInt(limitParam, 10, 32); err == nil {
			limit = int32(parsedLimit)
		}
	}

	offset := int32(0) // default
	if offsetParam != "" {
		if parsedOffset, err := strconv.ParseInt(offsetParam, 10, 32); err == nil {
			offset = int32(parsedOffset)
		}
	}

	blocks, err := con.etl.GetBlocks(c.Request().Context(), &connect.Request[v1.GetBlocksRequest]{
		Msg: &v1.GetBlocksRequest{
			Limit:  limit,
			Offset: offset,
		},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get blocks"})
	}

	return c.JSON(http.StatusOK, blocks.Msg)
}

// HTMX Fragment Handlers
func (con *Console) StatsHeaderFragment(c echo.Context) error {
	statsResp, err := con.etl.GetStats(c.Request().Context(), &connect.Request[v1.GetStatsRequest]{
		Msg: &v1.GetStatsRequest{},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get dashboard stats")
	}

	stats := &pages.DashboardStats{
		CurrentBlockHeight:  statsResp.Msg.CurrentBlockHeight,
		ChainID:             statsResp.Msg.ChainId,
		BPS:                 statsResp.Msg.Bps,
		IsSyncing:           statsResp.Msg.SyncStatus != nil && statsResp.Msg.SyncStatus.IsSyncing,
		LatestIndexedHeight: statsResp.Msg.SyncStatus.GetLatestIndexedHeight(),
		LatestChainHeight:   statsResp.Msg.SyncStatus.GetLatestChainHeight(),
		BlockDelta:          statsResp.Msg.SyncStatus.GetBlockDelta(),
	}

	fragment := pages.StatsHeaderFragment(stats)
	return fragment.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) NetworkSidebarFragment(c echo.Context) error {
	statsResp, err := con.etl.GetStats(c.Request().Context(), &connect.Request[v1.GetStatsRequest]{
		Msg: &v1.GetStatsRequest{},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get dashboard stats")
	}

	stats := &pages.DashboardStats{
		ValidatorCount:  statsResp.Msg.ValidatorCount,
		LatestBlock:     statsResp.Msg.LatestBlock,
		RecentProposers: statsResp.Msg.RecentProposers,
		IsSyncing:       statsResp.Msg.SyncStatus != nil && statsResp.Msg.SyncStatus.IsSyncing,
	}

	fragment := pages.NetworkSidebarFragment(stats)
	return fragment.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) TPSFragment(c echo.Context) error {
	statsResp, err := con.etl.GetStats(c.Request().Context(), &connect.Request[v1.GetStatsRequest]{
		Msg: &v1.GetStatsRequest{},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get dashboard stats")
	}

	stats := &pages.DashboardStats{
		TPS: statsResp.Msg.Tps,
	}

	fragment := pages.TPSFragment(stats)
	return fragment.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) TotalTransactionsFragment(c echo.Context) error {
	statsResp, err := con.etl.GetStats(c.Request().Context(), &connect.Request[v1.GetStatsRequest]{
		Msg: &v1.GetStatsRequest{},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get dashboard stats")
	}

	stats := &pages.DashboardStats{
		TotalTransactions: statsResp.Msg.TotalTransactions,
	}

	fragment := pages.TotalTransactionsFragment(stats)
	return fragment.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) LivePlaysSSE(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return c.String(http.StatusInternalServerError, "Streaming unsupported!")
	}

	rand.Seed(time.Now().UnixNano())

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		default:
			// Generate coordinates focused on North America
			// US mainland: roughly 25°N to 49°N, 125°W to 66°W
			// Include parts of Mexico (down to ~20°N) and Canada (up to ~55°N)
			lat := 20 + rand.Float64()*35   // 20°N to 55°N (Mexico to Canada)
			lng := -130 + rand.Float64()*65 // 130°W to 65°W (West coast to East coast)

			play := &pages.PlayEvent{
				Timestamp: time.Now().Format(time.RFC3339),
				Lat:       lat,
				Lng:       lng,
				Duration:  rand.Intn(3) + 2, // 2-4 seconds
			}

			jsonData, err := json.Marshal(play)
			if err != nil {
				return err
			}

			fmt.Fprintf(c.Response(), "data: %s\n\n", jsonData)
			flusher.Flush()

			time.Sleep(time.Second)
		}
	}
}
