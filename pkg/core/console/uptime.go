package console

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"connectrpc.com/connect"
	ethv1 "github.com/AudiusProject/audiusd/pkg/api/eth/v1"
	"github.com/AudiusProject/audiusd/pkg/core/console/views/pages"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

const (
	activeValidatorReportHistoryLength = 30
	validatorReportHistoryLength       = 5
)

func (cs *Console) uptimeFragment(c echo.Context) error {
	ctx := c.Request().Context()
	rollupNodeAddress := c.Param("validator")
	rollupBlockEnd := c.Param("rollup")

	if rollupNodeAddress == "" {
		rollupNodeAddress = cs.state.cometAddress
	}

	// Get active report
	activeReport, err := cs.getActiveSlaReport(ctx, rollupBlockEnd, rollupNodeAddress)
	if err != nil {
		cs.logger.Error("Falled to get active Proof Of Work report", "error", err)
		return err
	}

	// Attach report to this node
	myUptime := pages.NodeUptime{
		Address:       rollupNodeAddress,
		ActiveReport:  activeReport,
		ReportHistory: make([]pages.SlaReport, 0, 30),
	}

	// Get avg block time
	avgBlockTimeMs, err := cs.getAverageBlockTimeForReport(ctx, activeReport)
	if err != nil {
		cs.logger.Error("Failed to calculate average block time", "error", err)
		return err
	}

	allEndpointsResp, err := cs.eth.GetRegisteredEndpoints(
		ctx,
		connect.NewRequest(&ethv1.GetRegisteredEndpointsRequest{}),
	)
	if err != nil {
		cs.logger.Error("Falled to get all registered endpoints from eth service", "error", err)
		return err
	}

	// Gather validator info
	endpointMap := make(map[string]*pages.NodeUptime, len(allEndpointsResp.Msg.Endpoints))
	for _, ep := range allEndpointsResp.Msg.Endpoints {
		node, err := cs.db.GetNodeByEndpoint(ctx, ep.Endpoint)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			cs.logger.Error("Failed to get registered node from db", "endpoint", ep.Endpoint, "error", err)
			return err
		} else if errors.Is(err, pgx.ErrNoRows) {
			endpointMap[ep.Endpoint] = &pages.NodeUptime{
				Endpoint:      ep.Endpoint,
				Owner:         ep.Owner,
				IsValidator:   true,
				ReportHistory: make([]pages.SlaReport, 0, validatorReportHistoryLength),
			}
		} else {
			endpointMap[node.CometAddress] = &pages.NodeUptime{
				Endpoint:      ep.Endpoint,
				Owner:         ep.Owner,
				Address:       node.CometAddress,
				IsValidator:   true,
				ReportHistory: make([]pages.SlaReport, 0, validatorReportHistoryLength),
			}
		}
	}
	_, isValidator := endpointMap[rollupNodeAddress]
	myUptime.IsValidator = isValidator
	totalCometValidators, err := cs.db.TotalValidators(ctx)
	if err != nil {
		cs.logger.Error("Failed to get count of all validators from db", "error", err)
		return err
	}

	// Get history for this node
	recentRollups, err := cs.db.GetRecentRollupsForNode(
		ctx,
		db.GetRecentRollupsForNodeParams{
			Limit:   activeValidatorReportHistoryLength,
			Address: cs.state.cometAddress,
		},
	)
	if err != nil && err != pgx.ErrNoRows {
		cs.logger.Error("Failed to get recent rollups from db", "error", err)
		return err
	}
	for _, rr := range recentRollups {
		reportQuota := int32(0)
		if totalCometValidators > 0 {
			reportQuota = int32(rr.BlockEnd-rr.BlockStart) / int32(totalCometValidators)
		}
		myUptime.ReportHistory = append(
			myUptime.ReportHistory,
			pages.SlaReport{
				SlaRollupId:    rr.ID,
				TxHash:         rr.TxHash,
				BlockStart:     rr.BlockStart,
				BlockEnd:       rr.BlockEnd,
				BlocksProposed: rr.BlocksProposed.Int32,
				Quota:          reportQuota,
				Time:           rr.Time.Time,
			},
		)
	}

	// Get history for all validators
	bulkRecentReports, err := cs.db.GetRecentRollupsForAllNodes(
		ctx,
		db.GetRecentRollupsForAllNodesParams{
			ID:    activeReport.SlaRollupId,
			Limit: int32(validatorReportHistoryLength),
		},
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		cs.logger.Error("Failure getting bulk recent reports", "error", err)
		return err
	}
	for _, rr := range bulkRecentReports {
		if valData, ok := endpointMap[rr.Address.String]; ok {
			var reportQuota int32 = 0
			if totalCometValidators > 0 {
				reportQuota = int32(rr.BlockEnd-rr.BlockStart) / int32(totalCometValidators)
			}
			rep := pages.SlaReport{
				SlaRollupId:    rr.ID,
				TxHash:         rr.TxHash,
				BlockStart:     rr.BlockStart,
				BlockEnd:       rr.BlockEnd,
				BlocksProposed: rr.BlocksProposed.Int32,
				Quota:          reportQuota,
				Time:           rr.Time.Time,
			}
			if rr.ID == myUptime.ActiveReport.SlaRollupId {
				valData.ActiveReport = rep
			}
			valData.ReportHistory = append(valData.ReportHistory, rep)
		}
	}

	// Get proof of storage history
	posRollups, err := cs.db.GetStorageProofRollups(
		ctx,
		db.GetStorageProofRollupsParams{
			BlockHeight:   activeReport.BlockStart,
			BlockHeight_2: activeReport.BlockEnd,
		},
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		cs.logger.Error("Failure getting proof of storage rollups", "error", err)
		return err
	}
	for _, posr := range posRollups {
		if valData, ok := endpointMap[posr.Address]; ok {
			valData.ActiveReport.PoSChallengesFailed = int32(posr.FailedCount)
			valData.ActiveReport.PoSChallengesTotal = int32(posr.TotalCount)
		}
		if posr.Address == myUptime.Address {
			myUptime.ActiveReport.PoSChallengesFailed = int32(posr.FailedCount)
			myUptime.ActiveReport.PoSChallengesTotal = int32(posr.TotalCount)
		}
	}

	// Store validators as sorted slice
	// (adjust sorting method to fit display preference)
	sortedEndpoints := make([]*pages.NodeUptime, 0, len(endpointMap))
	for _, v := range endpointMap {
		sortedEndpoints = append(sortedEndpoints, v)
	}
	sort.Slice(sortedEndpoints, func(i, j int) bool {
		if sortedEndpoints[i].ActiveReport.BlocksProposed != sortedEndpoints[j].ActiveReport.BlocksProposed {
			return sortedEndpoints[i].ActiveReport.BlocksProposed < sortedEndpoints[j].ActiveReport.BlocksProposed
		}
		return sortedEndpoints[i].Endpoint < sortedEndpoints[j].Endpoint
	})

	return cs.views.RenderUptimeView(c, &pages.UptimePageView{
		ActiveNodeUptime: myUptime,
		ValidatorUptimes: sortedEndpoints,
		AvgBlockTimeMs:   avgBlockTimeMs,
	})
}

func (cs *Console) getActiveSlaReport(ctx context.Context, rollupBlockEnd, rollupNodeAddress string) (pages.SlaReport, error) {
	var report pages.SlaReport

	var rollup db.SlaRollup
	var err error
	if rollupBlockEnd == "" || rollupBlockEnd == "latest" {
		rollup, err = cs.db.GetLatestSlaRollup(ctx)
	} else if i, err := strconv.Atoi(rollupBlockEnd); err == nil {
		rollup, err = cs.db.GetSlaRollupWithBlockEnd(ctx, int64(i))
	} else {
		err = fmt.Errorf("Sla page called with invalid rollup block end %s", rollupBlockEnd)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		err = fmt.Errorf("Failed to retrieve SlaRollup from db: %v", err)
		return report, err
	}

	mySlaNodeReport, err := cs.db.GetRollupReportForNodeAndId(
		ctx,
		db.GetRollupReportForNodeAndIdParams{
			Address:     rollupNodeAddress,
			SlaRollupID: pgtype.Int4{Int32: rollup.ID, Valid: true},
		},
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		err = fmt.Errorf("Error while fetching this node's report for latest SlaRollup from db: %v", err)
		return report, err
	}

	var quota int32 = 0
	var numValidators int64 = 0
	numValidators, err = cs.db.TotalValidators(ctx)
	if err != nil {
		err = fmt.Errorf("Could not get total validators from db: %v", err)
		return report, err
	}
	if numValidators > int64(0) {
		quota = int32(rollup.BlockEnd-rollup.BlockStart) / int32(numValidators)
	}
	report = pages.SlaReport{
		SlaRollupId:    rollup.ID,
		TxHash:         rollup.TxHash,
		BlockStart:     rollup.BlockStart,
		BlockEnd:       rollup.BlockEnd,
		BlocksProposed: mySlaNodeReport.BlocksProposed,
		Quota:          quota,
		Time:           rollup.Time.Time,
	}

	return report, nil
}

func (cs *Console) getAverageBlockTimeForReport(ctx context.Context, report pages.SlaReport) (int, error) {
	var avgBlockTimeMs = 0
	previousRollup, err := cs.db.GetPreviousSlaRollupFromId(ctx, report.SlaRollupId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		err = fmt.Errorf("Failure reading previous SlaRollup from db: %v", err)
	} else if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	} else if err == nil && report.BlockEnd != 0 {
		totalBlocks := int(report.BlockEnd - report.BlockStart)
		avgBlockTimeMs = int(report.Time.UnixMilli()-previousRollup.Time.Time.UnixMilli()) / totalBlocks
	}
	return avgBlockTimeMs, err
}
