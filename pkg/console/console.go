package console

import (
	"context"
	"net/http"
	"strconv"
	"strings"

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
	e.GET("/validator/:address", con.stubRoute)

	e.GET("/blocks", con.Blocks)
	e.GET("/block/:height", con.Block)

	e.GET("/transactions", con.Transactions)
	e.GET("/transaction/:hash", con.stubRoute)

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
	p := pages.Dashboard()
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) Validators(c echo.Context) error {
	p := pages.Validators()
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) Blocks(c echo.Context) error {
	p := pages.Blocks()
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) Transactions(c echo.Context) error {
	p := pages.Transactions()
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
	p := pages.Block(uint64(height))
	return p.Render(c.Request().Context(), c.Response().Writer)
}

func (con *Console) stubRoute(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
