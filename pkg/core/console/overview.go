package console

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (cs *Console) overviewPage(c echo.Context) error {
	return c.Redirect(http.StatusMovedPermanently, "/console/uptime")
}
