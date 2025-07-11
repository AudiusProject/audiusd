package console

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/AudiusProject/audiusd/pkg/core/console/views/pages"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

const maxBlockRange = int64(1000)

func (cs *Console) posFragment(c echo.Context) error {
	ctx := c.Request().Context()
	start, end := cs.getValidBlockRange(ctx, c.QueryParam("block_start"), c.QueryParam("block_end"))

	proofs, err := cs.db.GetStorageProofsForNodeInRange(
		ctx,
		db.GetStorageProofsForNodeInRangeParams{
			BlockHeight:   start,
			BlockHeight_2: end,
			Address:       cs.state.cometAddress,
		},
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		err = fmt.Errorf("Failed to retrieve SlaRollup from db: %v", err)
		return err
	}

	validators, err := cs.db.GetAllRegisteredNodes(ctx)
	if err != nil && err != pgx.ErrNoRows {
		cs.logger.Error("Failed to get registered nodes from db", "error", err)
		return err
	}
	validatorMap := make(map[string]string, len(validators))
	for _, v := range validators {
		validatorMap[v.CometAddress] = v.Endpoint
	}

	pageProofs := make([]pages.StorageProof, 0, len(proofs))
	for _, p := range proofs {
		ep, ok := validatorMap[p.Address]
		if !ok {
			ep = ""
		}
		psp := pages.StorageProof{
			BlockHeight: p.BlockHeight,
			Endpoint:    ep,
			CID:         p.Cid.String,
			Status:      string(p.Status),
		}
		pageProofs = append(pageProofs, psp)
	}

	return cs.views.RenderPoSView(c, &pages.PoSPageView{
		Address:       cs.state.cometAddress,
		BlockStart:    start,
		BlockEnd:      end,
		StorageProofs: pageProofs,
	})
}

func (cs *Console) getValidBlockRange(ctx context.Context, startParam, endParam string) (int64, int64) {
	start := int64(0)
	end := int64(0)

	// default to last 'maxBlockRange' blocks
	if startParam == "" && endParam == "" {
		abciInfo, err := cs.state.rpc.ABCIInfo(ctx)
		if err != nil {
			cs.logger.Error("Could not get abciInfo for default block range")
			return start, end
		}
		return max(abciInfo.Response.LastBlockHeight-maxBlockRange, int64(0)), abciInfo.Response.LastBlockHeight
	}

	if i, err := strconv.Atoi(startParam); err == nil {
		start = max(int64(i), int64(0))
	}
	if j, err := strconv.Atoi(endParam); err == nil {
		end = max(int64(j), int64(0))
	}
	if end < start || end-start > maxBlockRange {
		start = max(end-maxBlockRange, int64(0))
	}
	return start, end
}
