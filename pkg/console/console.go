package console

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/console/templates/pages"
	"github.com/AudiusProject/audiusd/pkg/etl"
	"github.com/AudiusProject/audiusd/pkg/etl/db"
	"github.com/jackc/pgx/v5/pgtype"
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
	env    string
	e      *echo.Echo
	etl    *etl.ETLService
	logger *common.Logger
}

// TODO: Add these structs to a proper location once we have the full ETL API defined
type DashboardStats struct {
	CurrentBlockHeight           int64
	ChainID                      string
	BPS                          float64
	TPS                          float64
	TotalTransactions            int64
	ValidatorCount               int32
	LatestBlock                  *LatestBlockInfo
	RecentProposers              []*ProposerInfo
	IsSyncing                    bool
	LatestIndexedHeight          int64
	LatestChainHeight            int64
	BlockDelta                   int64
	TotalTransactions24h         int64
	TotalTransactionsPrevious24h int64
	TotalTransactions7d          int64
	TotalTransactions30d         int64
	AvgBlockTime                 float64
}

type LatestBlockInfo struct {
	// TODO: Define fields for latest block info
	Height   int64
	Proposer string
	Time     time.Time
	TxCount  int32
}

type ProposerInfo struct {
	// TODO: Define fields for proposer info
	Address     string
	BlockHeight int64
	Time        time.Time
}

type TransactionTypeBreakdown struct {
	Type  string
	Count int64
	Color string
}

// TODO: Add more placeholder types for features we haven't implemented yet
type PlayEvent struct {
	Timestamp string  `json:"timestamp"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Duration  int     `json:"duration"`
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
		if err := con.e.Start(":3000"); err != nil {
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
func (con *Console) getTransactionsWithBlockHeights(ctx context.Context, limit, offset int32) ([]*db.EtlTransaction, map[string]int64, error) {
	// Use GetTransactionsByPage for proper offset-based pagination
	transactions, err := con.etl.GetDB().GetTransactionsByPage(ctx, db.GetTransactionsByPageParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, nil, err
	}

	// Convert to pointers and create block heights map
	txPointers := make([]*db.EtlTransaction, len(transactions))
	blockHeights := make(map[string]int64)
	for i := range transactions {
		txPointers[i] = &transactions[i]
		blockHeights[transactions[i].TxHash] = transactions[i].BlockHeight
	}

	return txPointers, blockHeights, nil
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
	// TODO: Implement proper dashboard stats calculation using database queries

	// Get total transactions count
	totalTransactions, err := con.etl.GetDB().GetTotalTransactions(c.Request().Context())
	if err != nil {
		con.logger.Warn("Failed to get total transactions", "error", err)
		totalTransactions = 0
	}

	// Get latest indexed block
	latestBlockHeight, err := con.etl.GetDB().GetLatestIndexedBlock(c.Request().Context())
	if err != nil {
		con.logger.Warn("Failed to get latest block height", "error", err)
		latestBlockHeight = 0
	}

	// Get some recent transactions for the dashboard
	transactions, blockHeights, err := con.getTransactionsWithBlockHeights(c.Request().Context(), 10, 0)
	if err != nil {
		con.logger.Warn("Failed to get transactions", "error", err)
		transactions = []*db.EtlTransaction{}
		blockHeights = make(map[string]int64)
	}

	blocks, err := con.etl.GetDB().GetBlocksByPage(c.Request().Context(), db.GetBlocksByPageParams{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		con.logger.Warn("Failed to get blocks", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to get blocks")
	}

	blockPointers := make([]*db.EtlBlock, len(blocks))
	for i := range blocks {
		blockPointers[i] = &blocks[i]
	}

	// TODO: Implement these properly with database queries
	stats := &pages.DashboardStats{
		CurrentBlockHeight:           latestBlockHeight,
		ChainID:                      "audius-1", // TODO: Get from database or config
		BPS:                          0.5,        // TODO: Calculate from block times
		TPS:                          0.1,        // TODO: Calculate from transaction times
		TotalTransactions:            int64(totalTransactions),
		ValidatorCount:               0,     // TODO: Count active validators
		LatestBlock:                  nil,   // TODO: Implement
		RecentProposers:              nil,   // TODO: Implement
		IsSyncing:                    false, // TODO: Implement sync status check
		LatestIndexedHeight:          latestBlockHeight,
		LatestChainHeight:            latestBlockHeight,
		BlockDelta:                   0,
		TotalTransactions24h:         0,   // TODO: Implement time-based queries
		TotalTransactionsPrevious24h: 0,   // TODO: Implement time-based queries
		TotalTransactions7d:          0,   // TODO: Implement time-based queries
		TotalTransactions30d:         0,   // TODO: Implement time-based queries
		AvgBlockTime:                 2.0, // TODO: Calculate from actual block times
	}

	// TODO: Implement transaction breakdown calculation
	transactionBreakdown := []*pages.TransactionTypeBreakdown{
		{Type: "ManageEntity", Count: int64(totalTransactions), Color: "bg-blue-500"},
	}

	// Calculate sync progress percentage
	syncProgressPercentage := float64(100) // Assume synced for now

	props := pages.DashboardProps{
		Stats:                  stats,
		TransactionBreakdown:   transactionBreakdown,
		RecentBlocks:           blockPointers,
		RecentTransactions:     transactions,
		BlockHeights:           blockHeights,
		SyncProgressPercentage: syncProgressPercentage,
	}

	p := pages.Dashboard(props)

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

	// Default to active validators
	if queryType == "" {
		queryType = "active"
	}

	// Calculate offset from page number
	offset := (page - 1) * count

	var validators []*db.EtlValidator
	validatorUptimeMap := make(map[string][]*db.EtlSlaNodeReport)

	ctx := c.Request().Context()

	switch queryType {
	case "active":
		// Get active validators
		validatorsData, err := con.etl.GetDB().GetActiveValidators(ctx, db.GetActiveValidatorsParams{
			Limit:  count,
			Offset: offset,
		})
		if err != nil {
			con.logger.Warn("Failed to get active validators", "error", err)
			validatorsData = []db.EtlValidator{}
		}

		// Convert to pointers and apply endpoint filter
		for i := range validatorsData {
			if endpointFilter == "" || strings.Contains(strings.ToLower(validatorsData[i].Endpoint), strings.ToLower(endpointFilter)) {
				validators = append(validators, &validatorsData[i])

				// Get uptime data for each validator
				reports, err := con.etl.GetDB().GetSlaNodeReportsByAddress(ctx, db.GetSlaNodeReportsByAddressParams{
					Address: validatorsData[i].CometAddress,
					Limit:   5, // Get last 5 SLA reports
				})
				if err != nil {
					con.logger.Warn("Failed to get SLA reports", "address", validatorsData[i].CometAddress, "error", err)
				} else {
					reportPointers := make([]*db.EtlSlaNodeReport, len(reports))
					for j := range reports {
						reportPointers[j] = &reports[j]
					}
					validatorUptimeMap[validatorsData[i].CometAddress] = reportPointers
				}
			}
		}

	case "registrations":
		// Get validator registrations - this will need a different approach since it's a different table
		regsData, err := con.etl.GetDB().GetValidatorRegistrations(ctx, db.GetValidatorRegistrationsParams{
			Limit:  count,
			Offset: offset,
		})
		if err != nil {
			con.logger.Warn("Failed to get validator registrations", "error", err)
			regsData = []db.GetValidatorRegistrationsRow{}
		}

		// Convert registrations to validator format for template
		for i := range regsData {
			validator := &db.EtlValidator{
				ID:           regsData[i].ID,
				Address:      regsData[i].Address,
				Endpoint:     regsData[i].Endpoint, // Already a string
				CometAddress: regsData[i].CometAddress,
				NodeType:     regsData[i].NodeType,    // Already a string
				Spid:         regsData[i].Spid,        // Already a string
				VotingPower:  regsData[i].VotingPower, // Already int64
				Status:       "registered",
				RegisteredAt: regsData[i].BlockHeight,
				CreatedAt:    pgtype.Timestamp{Time: time.Now(), Valid: true}, // Manual timestamp
			}
			if endpointFilter == "" || strings.Contains(strings.ToLower(validator.Endpoint), strings.ToLower(endpointFilter)) {
				validators = append(validators, validator)
			}
		}

	case "deregistrations":
		// Get validator deregistrations
		deregsData, err := con.etl.GetDB().GetValidatorDeregistrations(ctx, db.GetValidatorDeregistrationsParams{
			Limit:  count,
			Offset: offset,
		})
		if err != nil {
			con.logger.Warn("Failed to get validator deregistrations", "error", err)
			deregsData = []db.GetValidatorDeregistrationsRow{}
		}

		// Convert deregistrations to validator format for template
		for i := range deregsData {
			endpoint := ""
			if deregsData[i].Endpoint.Valid {
				endpoint = deregsData[i].Endpoint.String
			}
			nodeType := ""
			if deregsData[i].NodeType.Valid {
				nodeType = deregsData[i].NodeType.String
			}
			spid := ""
			if deregsData[i].Spid.Valid {
				spid = deregsData[i].Spid.String
			}
			votingPower := int64(0)
			if deregsData[i].VotingPower.Valid {
				votingPower = deregsData[i].VotingPower.Int64
			}

			validator := &db.EtlValidator{
				ID:           deregsData[i].ID,
				Address:      "",
				Endpoint:     endpoint,
				CometAddress: deregsData[i].CometAddress,
				NodeType:     nodeType,
				Spid:         spid,
				VotingPower:  votingPower,
				Status:       "deregistered",
				RegisteredAt: deregsData[i].BlockHeight,
				CreatedAt:    pgtype.Timestamp{Time: time.Now(), Valid: true}, // placeholder
			}
			if endpointFilter == "" || strings.Contains(strings.ToLower(validator.Endpoint), strings.ToLower(endpointFilter)) {
				validators = append(validators, validator)
			}
		}
	}

	// Calculate pagination state
	hasNext := len(validators) == int(count) // Simple check - if we got the full limit, there might be more
	hasPrev := page > 1

	props := pages.ValidatorsProps{
		Validators:         validators,
		ValidatorUptimeMap: validatorUptimeMap,
		CurrentPage:        page,
		HasNext:            hasNext,
		HasPrev:            hasPrev,
		PageSize:           count,
		QueryType:          queryType,
		EndpointFilter:     endpointFilter,
	}

	p := pages.Validators(props)
	return p.Render(ctx, c.Response().Writer)
}

func (con *Console) Validator(c echo.Context) error {
	address := c.Param("address")
	if address == "" {
		return c.String(http.StatusBadRequest, "Validator address required")
	}

	// TODO: Implement validator lookup using database
	// For now, return not found
	return c.String(http.StatusNotFound, "Validator not found - TODO: Implement validator lookup")
}

func (con *Console) ValidatorsUptime(c echo.Context) error {
	// TODO: Implement validators uptime using database queries
	return c.String(http.StatusNotImplemented, "TODO: Implement validators uptime")
}

func (con *Console) ValidatorsUptimeByRollup(c echo.Context) error {
	// TODO: Implement validators uptime by rollup using database queries
	return c.String(http.StatusNotImplemented, "TODO: Implement validators uptime by rollup")
}

func (con *Console) Rollups(c echo.Context) error {
	// TODO: Implement rollups using database queries
	return c.String(http.StatusNotImplemented, "TODO: Implement rollups")
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

	// Get blocks from database
	blocksData, err := con.etl.GetDB().GetBlocksByPage(c.Request().Context(), db.GetBlocksByPageParams{
		Limit:  count,
		Offset: offset,
	})
	if err != nil {
		con.logger.Warn("Failed to get blocks", "error", err)
		blocksData = []db.EtlBlock{}
	}

	// Convert to pointers
	blocks := make([]*db.EtlBlock, len(blocksData))
	blockTransactions := make([]int32, len(blocksData))
	for i := range blocksData {
		blocks[i] = &blocksData[i]
		// Get transaction count for each block
		txCount, err := con.etl.GetDB().GetBlockTransactionCount(c.Request().Context(), blocksData[i].BlockHeight)
		if err != nil {
			con.logger.Warn("Failed to get transaction count for block", "height", blocksData[i].BlockHeight, "error", err)
			txCount = 0
		}
		blockTransactions[i] = int32(txCount)
	}

	// Calculate pagination state
	hasNext := len(blocks) == int(count) // Simple check - if we got the full limit, there might be more
	hasPrev := page > 1

	props := pages.BlocksProps{
		Blocks:            blocks,
		BlockTransactions: blockTransactions,
		CurrentPage:       page,
		HasNext:           hasNext,
		HasPrev:           hasPrev,
		PageSize:          count,
	}

	p := pages.Blocks(props)
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

	props := pages.TransactionsProps{
		Transactions: transactions,
		BlockHeights: blockHeights,
		CurrentPage:  page,
		HasNext:      hasNext,
		HasPrev:      hasPrev,
		PageSize:     count,
	}

	p := pages.Transactions(props)
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

	ctx := c.Request().Context()

	// Get block by height
	block, err := con.etl.GetDB().GetBlockByHeight(ctx, height)
	if err != nil {
		return c.String(http.StatusNotFound, fmt.Sprintf("Block not found at height %d", height))
	}

	// Get transactions for this block
	// First get all transactions and filter by block height
	// This is not the most efficient but will work for now - TODO: add GetTransactionsByBlockHeight query
	transactionsData, err := con.etl.GetDB().GetTransactionsByPage(ctx, db.GetTransactionsByPageParams{
		Limit:  1000, // Get a large number to ensure we get all for this block
		Offset: 0,
	})
	if err != nil {
		con.logger.Warn("Failed to get transactions", "error", err)
		transactionsData = []db.EtlTransaction{}
	}

	// Filter transactions for this specific block height
	var blockTransactions []*db.EtlTransaction
	for i := range transactionsData {
		if transactionsData[i].BlockHeight == height {
			blockTransactions = append(blockTransactions, &transactionsData[i])
		}
	}

	// Create block props
	props := pages.BlockProps{
		Block:        &block,
		Transactions: blockTransactions,
	}

	p := pages.Block(props)
	return p.Render(ctx, c.Response().Writer)
}

func (con *Console) Transaction(c echo.Context) error {
	txHash := c.Param("hash")
	if txHash == "" {
		return c.String(http.StatusBadRequest, "Transaction hash required")
	}

	ctx := c.Request().Context()

	// Get transaction by hash
	transaction, err := con.etl.GetDB().GetTransactionByHash(ctx, txHash)
	if err != nil {
		return c.String(http.StatusNotFound, fmt.Sprintf("Transaction not found: %s", txHash))
	}

	// Get block info for this transaction
	block, err := con.etl.GetDB().GetBlockByHeight(ctx, transaction.BlockHeight)
	if err != nil {
		con.logger.Warn("Failed to get block for transaction", "blockHeight", transaction.BlockHeight, "error", err)
		return c.String(http.StatusNotFound, fmt.Sprintf("Block not found at height %d", transaction.BlockHeight))
	}

	// Fetch transaction content based on type
	var content interface{}
	switch transaction.TxType {
	case "play":
		plays, err := con.etl.GetDB().GetPlaysByTxHash(ctx, txHash)
		if err != nil {
			con.logger.Warn("Failed to get plays for transaction", "txHash", txHash, "error", err)
		} else if len(plays) > 0 {
			// Convert to pointers for template
			playPointers := make([]*db.EtlPlay, len(plays))
			for i := range plays {
				playPointers[i] = &plays[i]
			}
			content = playPointers
		}

	case "manage_entity":
		entity, err := con.etl.GetDB().GetManageEntityByTxHash(ctx, txHash)
		if err != nil {
			con.logger.Warn("Failed to get manage entity for transaction", "txHash", txHash, "error", err)
		} else {
			content = &entity
		}

	case "validator_registration":
		registration, err := con.etl.GetDB().GetValidatorRegistrationByTxHash(ctx, txHash)
		if err != nil {
			con.logger.Warn("Failed to get validator registration for transaction", "txHash", txHash, "error", err)
		} else {
			content = &registration
		}

	case "validator_deregistration":
		deregistration, err := con.etl.GetDB().GetValidatorDeregistrationByTxHash(ctx, txHash)
		if err != nil {
			con.logger.Warn("Failed to get validator deregistration for transaction", "txHash", txHash, "error", err)
		} else {
			content = &deregistration
		}

	case "sla_rollup":
		slaRollup, err := con.etl.GetDB().GetSlaRollupByTxHash(ctx, txHash)
		if err != nil {
			con.logger.Warn("Failed to get SLA rollup for transaction", "txHash", txHash, "error", err)
		} else {
			content = &slaRollup
		}

	case "storage_proof":
		storageProof, err := con.etl.GetDB().GetStorageProofByTxHash(ctx, txHash)
		if err != nil {
			con.logger.Warn("Failed to get storage proof for transaction", "txHash", txHash, "error", err)
		} else {
			content = &storageProof
		}

	case "storage_proof_verification":
		storageProofVerification, err := con.etl.GetDB().GetStorageProofVerificationByTxHash(ctx, txHash)
		if err != nil {
			con.logger.Warn("Failed to get storage proof verification for transaction", "txHash", txHash, "error", err)
		} else {
			content = &storageProofVerification
		}
	}

	// Create transaction props
	props := pages.TransactionProps{
		Transaction: &transaction,
		Proposer:    block.ProposerAddress,
		Content:     content,
	}

	p := pages.Transaction(props)
	return p.Render(ctx, c.Response().Writer)
}

func (con *Console) Account(c echo.Context) error {
	address := c.Param("address")
	if address == "" {
		return c.String(http.StatusBadRequest, "Address parameter is required")
	}

	// TODO: Implement account lookup and transaction history using database queries
	return c.String(http.StatusNotImplemented, fmt.Sprintf("TODO: Implement account lookup for address %s", address))
}

func (con *Console) stubRoute(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

// HTMX Fragment Handlers
func (con *Console) StatsHeaderFragment(c echo.Context) error {
	// TODO: Implement stats header fragment using database queries
	return c.String(http.StatusNotImplemented, "TODO: Implement stats header fragment")
}

func (con *Console) NetworkSidebarFragment(c echo.Context) error {
	// TODO: Implement network sidebar fragment using database queries
	return c.String(http.StatusNotImplemented, "TODO: Implement network sidebar fragment")
}

func (con *Console) TPSFragment(c echo.Context) error {
	// TODO: Implement TPS fragment using database queries
	return c.String(http.StatusNotImplemented, "TODO: Implement TPS fragment")
}

func (con *Console) TotalTransactionsFragment(c echo.Context) error {
	// TODO: Implement total transactions fragment using database queries
	return c.String(http.StatusNotImplemented, "TODO: Implement total transactions fragment")
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

	// Subscribe to both block and play events from ETL pubsub
	blockCh := con.etl.GetBlockPubsub().Subscribe(etl.BlockTopic, 10)
	playCh := con.etl.GetPlayPubsub().Subscribe(etl.PlayTopic, 10)

	// Ensure cleanup on connection close
	defer func() {
		con.etl.GetBlockPubsub().Unsubscribe(etl.BlockTopic, blockCh)
		con.etl.GetPlayPubsub().Unsubscribe(etl.PlayTopic, playCh)
	}()

	// Throttle state for block events
	var (
		latestBlock    *db.EtlBlock
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

		case <-timeout:
			return nil

		case blockEvent := <-blockCh:
			if blockEvent != nil {
				latestBlock = blockEvent
			}

		case <-blockTicker.C:
			if latestBlock != nil && latestBlock.BlockHeight > lastSentHeight {
				// Send block event
				blockEvent := SSEEvent{
					Event: "block",
					Data: map[string]interface{}{
						"height":   latestBlock.BlockHeight,
						"proposer": latestBlock.ProposerAddress,
						"time":     latestBlock.BlockTime.Time.Format(time.RFC3339),
					},
				}
				eventData, _ := json.Marshal(blockEvent)
				fmt.Fprintf(c.Response(), "data: %s\n\n", string(eventData))
				lastSentHeight = latestBlock.BlockHeight
				flusher.Flush()
			}

		case play := <-playCh:
			if play != nil {
				// Get coordinates for the play location
				if play.City != "" && play.Region != "" && play.Country != "" {
					if latLong, err := con.etl.GetLocationDB().GetLatLong(c.Request().Context(), play.City, play.Region, play.Country); err == nil {
						lat := latLong.Latitude
						lng := latLong.Longitude
						// Send play event with coordinates
						playEvent := SSEEvent{
							Event: "play",
							Data: map[string]interface{}{
								"lat":       lat,
								"lng":       lng,
								"timestamp": time.Now().Format(time.RFC3339),
								"duration":  5, // Default 5 seconds for animation
							},
						}
						eventData, _ := json.Marshal(playEvent)
						fmt.Fprintf(c.Response(), "data: %s\n\n", string(eventData))
						flusher.Flush()
					}
				}
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

	// TODO: Implement search using database queries
	return c.JSON(http.StatusOK, map[string]interface{}{
		"results": []interface{}{},
	})
}
