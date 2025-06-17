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

	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:embed assets/css
var cssFS embed.FS

//go:embed assets/images
var imagesFS embed.FS

//go:embed assets/js
var jsFS embed.FS

type Console struct {
	env    string
	e      *echo.Echo
	etl    *etl.ETLService
	logger *common.Logger
}

func NewConsole(etl *etl.ETLService, e *echo.Echo, env string) *Console {
	if e == nil {
		e = echo.New()
	}
	if env == "" {
		env = "prod"
	}
	return &Console{etl: etl, e: e, logger: common.NewLogger(nil).Child("console"), env: env}
}

func (con *Console) SetupRoutes() {
	e := con.e
	e.HideBanner = true

	// Add environment context middleware
	envMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Add environment to the request context
			ctx := context.WithValue(c.Request().Context(), "env", con.env)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}

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

	// Apply middlewares
	e.Use(cacheControl)
	e.Use(envMiddleware)

	e.GET("/", con.Dashboard)
	e.GET("/hello", con.Hello)

	e.GET("/validators", con.Validators)
	e.GET("/validator/:address", con.Validator)
	e.GET("/validators/uptime", con.ValidatorsUptime)
	e.GET("/validators/uptime/:rollupid", con.ValidatorsUptimeByRollup)

	e.GET("/rollups", con.Rollups)

	e.GET("/blocks", con.Blocks)
	e.GET("/block/:height", con.Block)

	e.GET("/transactions", con.Transactions)
	e.GET("/transaction/:hash", con.Transaction)

	e.GET("/account/:address", con.Account)
	e.GET("/account/:address/transactions", con.stubRoute)
	e.GET("/account/:address/uploads", con.stubRoute)
	e.GET("/account/:address/releases", con.stubRoute)

	e.GET("/content", con.Content)
	e.GET("/content/:address", con.Content)

	e.GET("/release/:address", con.stubRoute)

	e.GET("/search", con.Search)

	// SSE endpoints
	e.GET("/sse/events", con.LiveEventsSSE)

	// HTMX Fragment routes
	e.GET("/fragments/tps", con.TPSFragment)
	e.GET("/fragments/total-transactions", con.TotalTransactionsFragment)
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

// getTransactionsWithBlockHeights is a helper method to get transactions with their block heights
func (con *Console) getTransactionsWithBlockHeights(ctx context.Context, limit, offset int32) ([]*v1.Block_Transaction, map[string]int64, error) {
	// Use the optimized method from ETL service that already gets block heights efficiently
	return con.etl.GetTransactionsWithBlockInfo(ctx, limit, offset)
}

func (con *Console) Hello(c echo.Context) error {
	param := "sup"
	if name := c.QueryParam("name"); name != "" {
		param = name
	}
	p := pages.Hello(param)

	// Use context with environment
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
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
	transactions, blockHeights, err := con.getTransactionsWithBlockHeights(c.Request().Context(), 10, 0)
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
		CurrentBlockHeight:           statsResp.Msg.CurrentBlockHeight,
		ChainID:                      statsResp.Msg.ChainId,
		BPS:                          statsResp.Msg.Bps,
		TPS:                          statsResp.Msg.Tps,
		TotalTransactions:            statsResp.Msg.TotalTransactions,
		ValidatorCount:               statsResp.Msg.ValidatorCount,
		LatestBlock:                  statsResp.Msg.LatestBlock,
		RecentProposers:              statsResp.Msg.RecentProposers,
		IsSyncing:                    statsResp.Msg.SyncStatus != nil && statsResp.Msg.SyncStatus.IsSyncing,
		LatestIndexedHeight:          statsResp.Msg.SyncStatus.GetLatestIndexedHeight(),
		LatestChainHeight:            statsResp.Msg.SyncStatus.GetLatestChainHeight(),
		BlockDelta:                   statsResp.Msg.SyncStatus.GetBlockDelta(),
		TotalTransactions24h:         statsResp.Msg.TotalTransactions_24H,
		TotalTransactionsPrevious24h: statsResp.Msg.TotalTransactionsPrevious_24H,
		TotalTransactions7d:          statsResp.Msg.TotalTransactions_7D,
		TotalTransactions30d:         statsResp.Msg.TotalTransactions_30D,
		AvgBlockTime:                 statsResp.Msg.AvgBlockTime,
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

	// Calculate sync progress percentage
	syncProgressPercentage := float64(0)
	if stats.LatestChainHeight > 0 {
		syncProgressPercentage = float64(stats.LatestIndexedHeight) / float64(stats.LatestChainHeight) * 100
	}

	p := pages.Dashboard(stats, transactionBreakdown, blocks.Msg.Blocks, transactions, blockHeights, syncProgressPercentage)

	// Use context with environment
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
}

func (con *Console) Validators(c echo.Context) error {
	// Parse query parameters
	pageParam := c.QueryParam("page")
	countParam := c.QueryParam("count")
	queryType := c.QueryParam("type") // "active", "registrations", "deregistrations"
	endpointFilter := c.QueryParam("endpoint_filter")

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

	// Add endpoint filter if specified
	if endpointFilter != "" {
		validatorsReq.EndpointFilter = &endpointFilter
	}

	validators, err := con.etl.GetValidators(c.Request().Context(), &connect.Request[v1.GetValidatorsRequest]{
		Msg: validatorsReq,
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get validators")
	}

	// Get lightweight uptime summary for all validators (just pass/fail status for recent rollups)
	// This is much faster than the full GetValidatorsUptime which fetches all detailed SLA data
	var validatorUptimeMap map[string][]*v1.SlaRollupScore
	uptimeResp, err := con.etl.GetValidatorsUptimeSummary(c.Request().Context(), &connect.Request[v1.GetValidatorsUptimeSummaryRequest]{
		Msg: &v1.GetValidatorsUptimeSummaryRequest{},
	})
	if err != nil {
		con.logger.Warn("Failed to get validators uptime summary", "error", err)
		// Continue without uptime data rather than fail the whole page
		validatorUptimeMap = make(map[string][]*v1.SlaRollupScore)
	} else {
		// Convert the ValidatorUptimeSummary array to a map for quick lookup
		// We need to convert the lightweight UptimeSummaryEntry to SlaRollupScore for template compatibility
		validatorUptimeMap = make(map[string][]*v1.SlaRollupScore)

		// Create a mapping from uptime validator address to rollups first
		uptimeDataMap := make(map[string][]*v1.SlaRollupScore)
		for _, validator := range uptimeResp.Msg.Validators {
			rollups := make([]*v1.SlaRollupScore, len(validator.RecentRollups))
			for i, entry := range validator.RecentRollups {
				// Convert from lightweight summary entry to full SlaRollupScore format
				// The database view pre-calculates the pass/fail status, so this is very efficient
				rollups[i] = &v1.SlaRollupScore{
					SlaRollupId:        entry.RollupId,
					BlocksProposed:     entry.BlocksProposed,
					BlockQuota:         entry.BlockQuota,
					ChallengesReceived: entry.ChallengesReceived,
					ChallengesFailed:   entry.ChallengesFailed,
					// Note: We have pre-calculated pass/fail in entry.Status ("pass", "fail", "offline", "unknown")
					// but the template calculates it from the metrics above, which is fine for compatibility
				}
			}
			uptimeDataMap[validator.ValidatorAddress] = rollups
			con.logger.Debug("Collected uptime data for validator", "validator_address", validator.ValidatorAddress, "rollup_count", len(rollups))
		}

		// Now map from the actual validators list using their CometAddress
		// This ensures we use the correct key format that the template expects
		for _, val := range validators.Msg.Validators {
			// Try to find uptime data for this validator
			// The uptime data should use comet addresses, so try both the comet address and regular address
			if rollups, exists := uptimeDataMap[val.CometAddress]; exists {
				validatorUptimeMap[val.CometAddress] = rollups
				con.logger.Debug("Mapped uptime data using CometAddress", "comet_address", val.CometAddress, "rollup_count", len(rollups))
			} else if rollups, exists := uptimeDataMap[val.Address]; exists {
				validatorUptimeMap[val.CometAddress] = rollups
				con.logger.Debug("Mapped uptime data using Address", "address", val.Address, "comet_address", val.CometAddress, "rollup_count", len(rollups))
			} else {
				con.logger.Debug("No uptime data found for validator", "address", val.Address, "comet_address", val.CometAddress)
			}
		}

		con.logger.Info("Retrieved validators uptime summary", "uptime_entries", len(uptimeResp.Msg.Validators), "mapped_entries", len(validatorUptimeMap))
	}

	// Calculate pagination state
	hasNext := validators.Msg.HasMore
	hasPrev := page > 1

	p := pages.Validators(validators.Msg.Validators, validatorUptimeMap, page, hasNext, hasPrev, count, queryType, endpointFilter)
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
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
		// try comet validator
		validator, err = con.etl.GetValidator(c.Request().Context(), &connect.Request[v1.GetValidatorRequest]{
			Msg: &v1.GetValidatorRequest{
				Identifier: &v1.GetValidatorRequest_CometAddress{CometAddress: strings.ToUpper(address)},
			},
		})
		if err != nil {
			return c.String(http.StatusNotFound, "Validator not found")
		}
	}

	// Get uptime data for this validator (last 12 rollups)
	var rollups []*v1.SlaRollupScore
	uptimeResp, err := con.etl.GetValidatorUptime(c.Request().Context(), &connect.Request[v1.GetValidatorUptimeRequest]{
		Msg: &v1.GetValidatorUptimeRequest{
			ValidatorAddress: validator.Msg.Validator.CometAddress, // Use comet address for uptime lookup
			Limit:            12,                                   // Last 12 rollups
		},
	})
	if err != nil {
		con.logger.Warn("Failed to get validator uptime data", "error", err, "validator", validator.Msg.Validator.CometAddress)
		// Continue without uptime data rather than fail the whole page
		rollups = []*v1.SlaRollupScore{}
	} else {
		rollups = uptimeResp.Msg.Rollups
		con.logger.Info("Retrieved validator uptime data", "validator", validator.Msg.Validator.CometAddress, "rollup_count", len(rollups))
		// Reverse the rollups slice so most recent appears on the right
		for i, j := 0, len(rollups)-1; i < j; i, j = i+1, j-1 {
			rollups[i], rollups[j] = rollups[j], rollups[i]
		}
	}

	p := pages.Validator(validator.Msg.Validator, validator.Msg.Events, rollups)
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
}

func (con *Console) ValidatorsUptime(c echo.Context) error {
	// Parse query parameters
	pageStr := c.QueryParam("page")
	countStr := c.QueryParam("count")

	var page int32 = 1
	var pageSize int32 = 20

	if pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			page = int32(p)
		}
	}

	if countStr != "" {
		if ps, err := strconv.ParseInt(countStr, 10, 32); err == nil && ps > 0 && ps <= 100 {
			pageSize = int32(ps)
		}
	}

	// Get rollups data
	rollupsResp, err := con.etl.GetSlaRollups(c.Request().Context(), &connect.Request[v1.GetSlaRollupsRequest]{
		Msg: &v1.GetSlaRollupsRequest{
			Page:     page,
			PageSize: pageSize,
		},
	})
	if err != nil {
		con.logger.Error("Failed to get rollups data", "error", err)
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to get rollups data: %v", err))
	}

	// Use the same Rollups template but with a different title context
	p := pages.UptimeRollups(
		rollupsResp.Msg.Rollups,
		rollupsResp.Msg.CurrentPage,
		rollupsResp.Msg.HasNext,
		rollupsResp.Msg.HasPrev,
		pageSize,
		rollupsResp.Msg.TotalCount,
	)
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
}

func (con *Console) ValidatorsUptimeByRollup(c echo.Context) error {
	rollupIdStr := c.Param("rollupid")
	if rollupIdStr == "" {
		return c.String(http.StatusBadRequest, "Rollup ID required")
	}

	rollupId, err := strconv.ParseInt(rollupIdStr, 10, 32)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid rollup ID")
	}

	// Get uptime data for specific rollup
	uptimeResp, err := con.etl.GetValidatorsUptimeByRollup(c.Request().Context(), &connect.Request[v1.GetValidatorsUptimeByRollupRequest]{
		Msg: &v1.GetValidatorsUptimeByRollupRequest{
			RollupId: int32(rollupId),
		},
	})
	if err != nil {
		con.logger.Error("Failed to get validator uptime data for rollup", "error", err, "rollup_id", rollupId)
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to get validator uptime data for rollup %d: %v", rollupId, err))
	}

	p := pages.ValidatorsUptimeByRollup(uptimeResp.Msg.Validators, int32(rollupId))
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
}

func (con *Console) Rollups(c echo.Context) error {
	// Parse query parameters
	pageStr := c.QueryParam("page")
	countStr := c.QueryParam("count")

	var page int32 = 1
	var pageSize int32 = 20

	if pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			page = int32(p)
		}
	}

	if countStr != "" {
		if ps, err := strconv.ParseInt(countStr, 10, 32); err == nil && ps > 0 && ps <= 100 {
			pageSize = int32(ps)
		}
	}

	// Get rollups data
	rollupsResp, err := con.etl.GetSlaRollups(c.Request().Context(), &connect.Request[v1.GetSlaRollupsRequest]{
		Msg: &v1.GetSlaRollupsRequest{
			Page:     page,
			PageSize: pageSize,
		},
	})
	if err != nil {
		con.logger.Error("Failed to get rollups data", "error", err)
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to get rollups data: %v", err))
	}

	con.logger.Info("Successfully retrieved rollups data", "rollup_count", len(rollupsResp.Msg.Rollups), "page", page)

	p := pages.Rollups(
		rollupsResp.Msg.Rollups,
		rollupsResp.Msg.CurrentPage,
		rollupsResp.Msg.HasNext,
		rollupsResp.Msg.HasPrev,
		pageSize,
		rollupsResp.Msg.TotalCount,
	)
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
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
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
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

	transactions, blockHeights, err := con.getTransactionsWithBlockHeights(c.Request().Context(), count, offset)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get transactions")
	}

	// Calculate pagination state
	hasNext := len(transactions) == int(count) // Simple check - if we got the full limit, there might be more
	hasPrev := page > 1

	p := pages.Transactions(transactions, blockHeights, page, hasNext, hasPrev, count)
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
}

func (con *Console) Content(c echo.Context) error {
	p := pages.Content()
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
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
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
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
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
}

func (con *Console) Account(c echo.Context) error {
	address := c.Param("address")
	if address == "" {
		return c.String(http.StatusBadRequest, "Address parameter is required")
	}

	// Parse query parameters for pagination
	pageStr := c.QueryParam("page")
	countStr := c.QueryParam("count")
	relationFilter := c.QueryParam("relation") // Get relation filter from query param
	startDateStr := c.QueryParam("start_date") // Get start date from query param
	endDateStr := c.QueryParam("end_date")     // Get end date from query param

	page := int32(1)
	if pageStr != "" {
		if parsedPage, err := strconv.ParseInt(pageStr, 10, 32); err == nil && parsedPage > 0 {
			page = int32(parsedPage)
		}
	}

	count := int32(50)
	if countStr != "" {
		if parsedCount, err := strconv.ParseInt(countStr, 10, 32); err == nil && parsedCount > 0 {
			count = int32(parsedCount)
		}
	}

	// Parse date parameters
	var startDate, endDate *timestamppb.Timestamp
	if startDateStr != "" {
		if parsedTime, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = timestamppb.New(parsedTime)
		}
	}
	if endDateStr != "" {
		if parsedTime, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// Set to end of day (23:59:59.999)
			endOfDay := parsedTime.Add(24*time.Hour - time.Nanosecond)
			endDate = timestamppb.New(endOfDay)
		}
	}

	// Calculate offset from page number
	offset := (page - 1) * count

	// Get transactions for this address from ETL service
	response, err := con.etl.GetTransactionsByAddress(c.Request().Context(), &connect.Request[v1.GetTransactionsByAddressRequest]{
		Msg: &v1.GetTransactionsByAddressRequest{
			Address: address,
			Limit:   count,
			Offset:  offset,
			RelationFilter: func() *string {
				if relationFilter != "" && relationFilter != "all" {
					return &relationFilter
				}
				return nil
			}(),
			StartDate: startDate,
			EndDate:   endDate,
		},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get transactions for address")
	}

	// Get available relation types for this address to populate the dropdown
	relationTypesResponse, err := con.etl.GetRelationTypesByAddress(c.Request().Context(), &connect.Request[v1.GetRelationTypesByAddressRequest]{
		Msg: &v1.GetRelationTypesByAddressRequest{
			Address: address,
		},
	})
	var relationTypes []string
	if err == nil {
		relationTypes = relationTypesResponse.Msg.RelationTypes
	}

	// Calculate pagination state
	hasNext := response.Msg.HasMore
	hasPrev := page > 1

	p := pages.Account(address, response.Msg.Transactions, page, hasNext, hasPrev, count, relationTypes, relationFilter, startDateStr, endDateStr)
	ctx := c.Request().Context()
	return p.Render(ctx, c.Response().Writer)
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

	// Calculate exact sync progress percentage
	var syncProgressPercentage float64
	if statsResp.Msg.SyncStatus != nil && statsResp.Msg.SyncStatus.GetLatestChainHeight() > 0 {
		syncProgressPercentage = float64(statsResp.Msg.SyncStatus.GetLatestIndexedHeight()) / float64(statsResp.Msg.SyncStatus.GetLatestChainHeight()) * 100
	}

	stats := &pages.DashboardStats{
		CurrentBlockHeight:  statsResp.Msg.CurrentBlockHeight,
		ChainID:             statsResp.Msg.ChainId,
		BPS:                 statsResp.Msg.Bps,
		IsSyncing:           statsResp.Msg.SyncStatus != nil && statsResp.Msg.SyncStatus.IsSyncing,
		LatestIndexedHeight: statsResp.Msg.SyncStatus.GetLatestIndexedHeight(),
		LatestChainHeight:   statsResp.Msg.SyncStatus.GetLatestChainHeight(),
		BlockDelta:          statsResp.Msg.SyncStatus.GetBlockDelta(),
		AvgBlockTime:        statsResp.Msg.AvgBlockTime,
	}

	fragment := pages.StatsHeaderFragment(stats, syncProgressPercentage)
	ctx := c.Request().Context()
	return fragment.Render(ctx, c.Response().Writer)
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
	ctx := c.Request().Context()
	return fragment.Render(ctx, c.Response().Writer)
}

func (con *Console) TPSFragment(c echo.Context) error {
	statsResp, err := con.etl.GetStats(c.Request().Context(), &connect.Request[v1.GetStatsRequest]{
		Msg: &v1.GetStatsRequest{},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get dashboard stats")
	}

	stats := &pages.DashboardStats{
		TPS:                  statsResp.Msg.Tps,
		TotalTransactions30d: statsResp.Msg.TotalTransactions_30D,
	}

	fragment := pages.TPSFragment(stats)
	ctx := c.Request().Context()
	return fragment.Render(ctx, c.Response().Writer)
}

func (con *Console) TotalTransactionsFragment(c echo.Context) error {
	statsResp, err := con.etl.GetStats(c.Request().Context(), &connect.Request[v1.GetStatsRequest]{
		Msg: &v1.GetStatsRequest{},
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get dashboard stats")
	}

	stats := &pages.DashboardStats{
		TotalTransactions:            statsResp.Msg.TotalTransactions,
		TotalTransactions24h:         statsResp.Msg.TotalTransactions_24H,
		TotalTransactionsPrevious24h: statsResp.Msg.TotalTransactionsPrevious_24H,
	}

	fragment := pages.TotalTransactionsFragment(stats)
	ctx := c.Request().Context()
	return fragment.Render(ctx, c.Response().Writer)
}

type SSEEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

const sseConnectionTTL = 1 * time.Minute

func (con *Console) LiveEventsSSE(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return nil
	}

	flusher.Flush()

	// Subscribe to play events from ETL pubsub
	playChannel := con.etl.GetPlayPubsub().Subscribe(etl.PlayTopic, 100)
	defer con.etl.GetPlayPubsub().Unsubscribe(etl.PlayTopic, playChannel)

	blockChannel := con.etl.GetBlockPubsub().Subscribe(etl.BlockTopic, 100)
	defer con.etl.GetBlockPubsub().Unsubscribe(etl.BlockTopic, blockChannel)

	// Throttle state for block events
	var (
		latestBlock    *v1.Block
		lastSentHeight int64
		blockTicker    = time.NewTicker(1 * time.Second)
	)
	defer blockTicker.Stop()

	flusher.Flush()

	timeout := time.After(sseConnectionTTL)

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case blockEvent := <-blockChannel:
			if blockEvent != nil {
				latestBlock = blockEvent
			}
		case <-timeout:
			return nil
		case <-blockTicker.C:
			if latestBlock != nil && latestBlock.Height > lastSentHeight {
				resp := &v1.StreamResponse_StreamBlocksResponse{
					Height:   latestBlock.Height,
					Proposer: latestBlock.Proposer,
				}

				event := SSEEvent{
					Event: "block",
					Data:  resp,
				}

				jsonData, err := json.Marshal(event)
				if err != nil {
					continue
				}
				fmt.Fprintf(c.Response(), "data: %s\n\n", jsonData)
				lastSentHeight = latestBlock.Height
				flusher.Flush()
			}

		case playEvent := <-playChannel:
			if playEvent != nil && playEvent.Latitude != 0 && playEvent.Longitude != 0 {
				// Convert ETL TrackPlay to PlayEvent format
				play := &pages.PlayEvent{
					Timestamp: playEvent.PlayedAt.AsTime().Format(time.RFC3339),
					Lat:       playEvent.Latitude,
					Lng:       playEvent.Longitude,
					Duration:  rand.Intn(3) + 2, // Keep random duration for animation (2-4 seconds)
				}

				event := SSEEvent{
					Event: "play",
					Data:  play,
				}

				jsonData, err := json.Marshal(event)
				if err != nil {
					con.logger.Error("Failed to marshal play event", "error", err)
					continue
				}

				fmt.Fprintf(c.Response(), "data: %s\n\n", jsonData)
				flusher.Flush()
			}
		}
	}
}

func (con *Console) Search(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"results": []interface{}{},
		})
	}

	// Call the ETL service search
	response, err := con.etl.Search(c.Request().Context(), &connect.Request[v1.SearchRequest]{
		Msg: &v1.SearchRequest{
			Query: query,
			Limit: 20,
		},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Search failed",
		})
	}

	// Convert protobuf results to JSON structure expected by frontend
	results := make([]map[string]interface{}, len(response.Msg.Results))
	for i, result := range response.Msg.Results {
		results[i] = map[string]interface{}{
			"id":       result.Id,
			"title":    result.Title,
			"subtitle": result.Subtitle,
			"type":     result.Type,
			"url":      result.Url,
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"results": results,
	})
}
