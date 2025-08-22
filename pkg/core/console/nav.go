package console

import (
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/labstack/echo/v4"
)

func (con *Console) navChainData(c echo.Context) error {
	res, err := con.core.GetNodeInfo(c.Request().Context(), &connect.Request[v1.GetNodeInfoRequest]{})
	if err != nil {
		return err
	}

	totalBlocks := fmt.Sprint(res.Msg.CurrentHeight)
	totalTxs := fmt.Sprint(res.Msg.TotalTransactions)
	isSyncing := !res.Msg.Synced

	return con.views.RenderNavChainData(c, totalBlocks, totalTxs, isSyncing)
}
