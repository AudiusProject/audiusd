package console

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/etl/v1"
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
	e   *echo.Echo
	etl *etl.ETLService
}

func NewConsole(etl *etl.ETLService) *Console {
	return &Console{etl: etl, e: echo.New()}
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
	// Get some recent blocks for the dashboard
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

	// Create mock dashboard stats
	stats := &pages.DashboardStats{
		CurrentBlockHeight: 0,
		ChainID:            "audius-1",
		BPS:                1.5,
		TPS:                12.3,
		TotalTransactions:  450123,
		ValidatorCount:     25,
		LatestBlock:        nil,
		RecentProposers: []string{
			"cosmos1abc123def456ghi789jkl012mno345pqr678stu",
			"cosmos1xyz987wvu654tsq321pon098mlk765ihg432fed",
			"cosmos1qwe456rty789uio012asd345fgh678jkl901zxc",
			"cosmos1mnb567vcx890qaz123wsx456edc789rfv012tgb",
		},
	}

	// If we have blocks, update stats with real data
	if len(blocks.Msg.Blocks) > 0 {
		latestBlock := blocks.Msg.Blocks[0]
		stats.CurrentBlockHeight = latestBlock.Height
		stats.LatestBlock = latestBlock

		// Extract recent proposers from blocks
		proposers := make([]string, 0, 4)
		for i, block := range blocks.Msg.Blocks {
			if i >= 4 {
				break
			}
			proposers = append(proposers, block.Proposer)
		}
		stats.RecentProposers = proposers
	}

	// Create mock transaction type breakdown
	transactionBreakdown := []*pages.TransactionTypeBreakdown{
		{Type: "Transfer", Count: 15420, Color: "bg-blue-500"},
		{Type: "Delegate", Count: 8934, Color: "bg-green-500"},
		{Type: "Vote", Count: 5123, Color: "bg-yellow-500"},
		{Type: "Register", Count: 2451, Color: "bg-purple-500"},
		{Type: "Other", Count: 1876, Color: "bg-gray-500"},
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

func (con *Console) APITransactions(c echo.Context) error {
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

	transactions, blockHeights, err := con.etl.GetTransactionsForAPI(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get transactions"})
	}

	// Create response with block heights for frontend
	response := map[string]interface{}{
		"transactions": make([]map[string]interface{}, len(transactions.Transactions)),
		"has_more":     transactions.HasMore,
		"total_count":  transactions.TotalCount,
	}

	for i, tx := range transactions.Transactions {
		blockHeight := int64(0)
		if bh, exists := blockHeights[tx.Hash]; exists {
			blockHeight = bh
		}

		response["transactions"].([]map[string]interface{})[i] = map[string]interface{}{
			"hash":         tx.Hash,
			"type":         tx.Type,
			"block_height": blockHeight,
			"timestamp":    tx.Timestamp.AsTime().Format(time.RFC3339Nano),
		}
	}

	return c.JSON(http.StatusOK, response)
}
