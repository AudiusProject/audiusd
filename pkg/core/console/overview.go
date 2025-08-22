package console

import (
	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/labstack/echo/v4"
)

func (cs *Console) overviewPage(c echo.Context) error {
	res, err := cs.core.GetStatus(c.Request().Context(), &connect.Request[v1.GetStatusRequest]{})
	if err != nil {
		return err
	}
	return cs.views.RenderOverview(c, res.Msg)
}
