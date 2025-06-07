package console

import (
	"context"
	"net/http"

	"github.com/AudiusProject/audiusd/pkg/etl"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"
)

type Console struct {
	e   *echo.Echo
	etl *etl.ETLService
}

func NewConsole(etl *etl.ETLService) *Console {
	return &Console{etl: etl}
}

func (con *Console) SetupRoutes() {
	e := con.e

	e.GET("/", con.stubRoute)

	e.GET("/validators", con.stubRoute)
	e.GET("/validator/:address", con.stubRoute)

	e.GET("/blocks", con.stubRoute)
	e.GET("/block/:height", con.stubRoute)

	e.GET("/transactions", con.stubRoute)
	e.GET("/transaction/:hash", con.stubRoute)

	e.GET("/account/:address", con.stubRoute)
	e.GET("/account/:address/transactions", con.stubRoute)
	e.GET("/account/:address/uploads", con.stubRoute)
	e.GET("/account/:address/releases", con.stubRoute)

	e.GET("/upload/:address", con.stubRoute)

	e.GET("/release/:address", con.stubRoute)
}

func (con *Console) Run() error {
	g, _ := errgroup.WithContext(context.Background())

	g.Go(func() error {
		return con.etl.Run()
	})

	g.Go(func() error {
		return con.e.Start(":8080")
	})

	return g.Wait()
}

func (con *Console) stubRoute(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
