package console

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

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
	slaMeetsThreshold                  = 0.8
	slaMissThreshold                   = 0.4
)

func (cs *Console) uptimeFragment(c echo.Context) error {
	ctx := c.Request().Context()
	endpointURL := c.Param("endpoint")
	rollupBlockEnd := c.Param("rollup")
	if endpointURL == "" {
		endpointURL = cs.config.NodeEndpoint
	} else {
		endpointURL = fmt.Sprintf("https://%s", endpointURL)
	}

	endpoints, activeEndpoint, err := cs.generateEndpoints(ctx, endpointURL)
	if err != nil {
		cs.logger.Error("failed to generate endpoints: %v", err)
		return err
	}

	if err := cs.populateSlaReportsForEndpoints(ctx, endpoints, activeEndpoint, rollupBlockEnd); err != nil {
		cs.logger.Error("failed to populate SLA reports for endpoints: %v", err)
		return err
	}

	// SLA rollup not found, abort early with empty data
	if activeEndpoint.ActiveReport == nil {
		return cs.views.RenderUptimeView(c, &pages.UptimePageView{})
	}

	avgBlockTimeMs, err := cs.getAverageBlockTimeForReport(ctx, activeEndpoint.ActiveReport)
	if err != nil {
		cs.logger.Error("Failed to calculate average block time", "error", err)
		return err
	}

	// Store endpoints as sorted slice
	// (adjust sorting method to fit display preference)
	sort.Slice(endpoints, func(i, j int) bool {
		// TODO incorporate exempt status
		if endpoints[i].ActiveReport.BlocksProposed != endpoints[j].ActiveReport.BlocksProposed {
			return endpoints[i].ActiveReport.BlocksProposed < endpoints[j].ActiveReport.BlocksProposed
		}
		return endpoints[i].Endpoint < endpoints[j].Endpoint
	})

	return cs.views.RenderUptimeView(c, &pages.UptimePageView{
		ActiveEndpoint: activeEndpoint,
		Endpoints:      endpoints,
		AvgBlockTimeMs: avgBlockTimeMs,
	})

	/*

		// 1. Get selected SLA rollup
		activeRollup, err := cs.getActiveSlaRollup(ctx, rollupBlockEnd)
		if err != nil {
			cs.logger.Error("failed to active sla rollup", "error", err)
			return err
		}

		// 2. Get selected endpoint
		activeEndpoint, err := cs.getEndpoint(ctx, endpoint)
		if err != nil {
			cs.logger.Error("failed to active endpoint", "error", err)
			return err
		}

		// 3. Get SLA Rollups around the selected SLA rollup time period
		dbReports, err := cs.db.GetRollupReportsForNodeInTimeRange(
			ctx,
			db.GetRollupReportsForNodeInTimeRangeParams{
				Address: activeEndpoint.CometAddress, // Does not matter if unset
				Time:    cs.db.ToPgxTimestamp(activeRollup.Time.Time.Add(-24 * time.Hour)),
				Time_2:  cs.db.ToPgxTimestamp(activeRollup.Time.Time.Add(24 * time.Hour)),
			},
		)
		if err != nil {
			cs.logger.Error("failed to get rollup reports", "error", err)
			return err
		}

		// 4. Apply each report to the endpoint's history
		pageReports := make([]*pages.SlaReport, len(dbReports))
		for i, dbrep := range dbReports {
			pagerep := &pages.SlaReport{
				SlaRollupId:    dbrep.ID,
				TxHash:         dbrep.TxHash,
				BlockStart:     dbrep.BlockStart,
				BlockEnd:       dbrep.BlockEnd,
				BlocksProposed: dbrep.BlocksProposed.Int32,
				Time:           dbrep.Time.Time,
			}

			if dbrep.ID == activeRollup.ID {
				activeEndpoint.ActiveReport = pagerep
			}

			if endpointIsExemptForSlaReport(activeEndpoint, dbrep.Time.Time, dbrep.Address.Valid) {
				pagerep.Status = pages.SlaExempt
				pagerep.Quota = 0
			} else {
				if dbrep.ValidatorCount == 0 {
					// unexpected state, but protect against divide-by-zero panic anyway
					dbrep.ValidatorCount += 1
				}
				quota := int32(dbrep.BlockEnd-dbrep.BlockStart) / int32(dbrep.ValidatorCount)
				if quota == 0 {
					// protect against divide-by-zero panic again
					quota += 1
				}
				pagerep.Quota = quota
				faultRatio := float64(dbrep.BlocksProposed.Int32) / float64(quota)
				if faultRatio < slaMeetsThreshold && faultRatio > 0 {
					pagerep.Status = pages.SlaPartial
				} else if faultRatio == 0 {
					pagerep.Status = pages.SlaDead
				} else {
					pagerep.Status = pages.SlaMet
				}
			}
			pageReports[i] = pagerep
		}

		// 5. Now do the same for all endpoints

		// ***********************************
		// old code
		// ***********************************

		rollupNodeAddress := cs.state.cometAddress

		// Get active report
		activeReport, err := cs.getActiveSlaReport(ctx, rollupBlockEnd, rollupNodeAddress)
		if err != nil {
			cs.logger.Error("Falled to get active Proof Of Work report", "error", err)
			return err
		}

		// Attach report to this node
		myUptime := pages.Endpoint{
			CometAddress: rollupNodeAddress,
			ActiveReport: activeReport,
			SlaReports:   make([]*pages.SlaReport, 0, 30),
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
		endpointMap := make(map[string]*pages.Endpoint, len(allEndpointsResp.Msg.Endpoints))
		for _, ep := range allEndpointsResp.Msg.Endpoints {
			node, err := cs.db.GetNodeByEndpoint(ctx, ep.Endpoint)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				cs.logger.Error("Failed to get registered node from db", "endpoint", ep.Endpoint, "error", err)
				return err
			} else if errors.Is(err, pgx.ErrNoRows) {
				endpointMap[ep.Endpoint] = &pages.Endpoint{
					Endpoint:   ep.Endpoint,
					Owner:      ep.Owner,
					SlaReports: make([]*pages.SlaReport, 0, validatorReportHistoryLength),
				}
			} else {
				endpointMap[node.CometAddress] = &pages.Endpoint{
					Endpoint:     ep.Endpoint,
					Owner:        ep.Owner,
					CometAddress: node.CometAddress,
					SlaReports:   make([]*pages.SlaReport, 0, validatorReportHistoryLength),
				}
			}
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
			myUptime.SlaReports = append(
				myUptime.SlaReports,
				&pages.SlaReport{
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
				rep := &pages.SlaReport{
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
				valData.SlaReports = append(valData.SlaReports, rep)
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
			if posr.Address == myUptime.CometAddress {
				myUptime.ActiveReport.PoSChallengesFailed = int32(posr.FailedCount)
				myUptime.ActiveReport.PoSChallengesTotal = int32(posr.TotalCount)
			}
		}

		// Store validators as sorted slice
		// (adjust sorting method to fit display preference)
		sortedEndpoints := make([]*pages.Endpoint, 0, len(endpointMap))
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
			ActiveEndpoint: activeEndpoint,
			Endpoints:      sortedEndpoints,
			AvgBlockTimeMs: avgBlockTimeMs,
		})
	*/
}

func (cs *Console) getActiveSlaRollup(ctx context.Context, rollupBlockEndParam string) (db.SlaRollup, error) {
	var rollup db.SlaRollup
	var err error
	if rollupBlockEndParam == "" || rollupBlockEndParam == "latest" {
		rollup, err = cs.db.GetLatestSlaRollup(ctx)
	} else if i, err := strconv.Atoi(rollupBlockEndParam); err == nil {
		rollup, err = cs.db.GetSlaRollupWithBlockEnd(ctx, int64(i))
	} else {
		err = fmt.Errorf("Sla page called with invalid rollup block end %s", rollupBlockEndParam)
	}
	if err != nil {
		err = fmt.Errorf("Failed to retrieve SlaRollup from db: %v", err)
		return rollup, err
	}

	return rollup, nil
}

func (cs *Console) getEndpoint(ctx context.Context, endpoint string) (*pages.Endpoint, error) {
	infoResp, err := cs.eth.GetRegisteredEndpointInfo(
		ctx,
		connect.NewRequest(&ethv1.GetRegisteredEndpointInfoRequest{
			Endpoint: endpoint,
		}),
	)
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			if connectErr.Code() == connect.CodeNotFound {
				return &pages.Endpoint{Endpoint: endpoint}, nil
			}
		}
		return nil, err
	}

	ep := &pages.Endpoint{
		Endpoint:        endpoint,
		EthAddress:      infoResp.Msg.Se.DelegateWallet,
		Owner:           infoResp.Msg.Se.Owner,
		RegisteredAt:    infoResp.Msg.Se.RegisteredAt.AsTime(),
		IsEthRegistered: true,
	}

	if validator, err := cs.db.GetRegisteredNodeByEthAddress(ctx, infoResp.Msg.Se.DelegateWallet); err != nil {
		ep.CometAddress = validator.CometAddress
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	return ep, nil
}

func (cs *Console) getActiveSlaReport(ctx context.Context, rollupBlockEnd, rollupNodeAddress string) (*pages.SlaReport, error) {
	var report *pages.SlaReport

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
	report = &pages.SlaReport{
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

func (cs *Console) getAverageBlockTimeForReport(ctx context.Context, report *pages.SlaReport) (int, error) {
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

func endpointIsExemptForSlaReport(endpoint *pages.Endpoint, reportTimestamp time.Time) bool {
	return !endpoint.IsEthRegistered || endpoint.RegisteredAt.After(reportTimestamp)
}

func (cs *Console) generateEndpoints(ctx context.Context, activeEndpointURL string) (endpoints []*pages.Endpoint, activeEndpoint *pages.Endpoint, err error) {
	// get all endpoints registered on eth
	endpointsResp, err := cs.eth.GetRegisteredEndpoints(
		ctx,
		connect.NewRequest(&ethv1.GetRegisteredEndpointsRequest{}),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error requesting endpoints from eth service: %v", err)
	}

	// transform endpoints into pages.Endpoint objects
	endpointMap := make(map[string]*pages.Endpoint, len(endpointsResp.Msg.Endpoints))
	for _, ep := range endpointsResp.Msg.Endpoints {
		pep := &pages.Endpoint{
			Endpoint:        ep.Endpoint,
			EthAddress:      ep.DelegateWallet,
			Owner:           ep.Owner,
			RegisteredAt:    ep.RegisteredAt.AsTime(),
			IsEthRegistered: true,
		}
		if ep.Endpoint == activeEndpointURL {
			activeEndpoint = pep
		}
		endpointMap[ep.Endpoint] = pep
	}
	if activeEndpoint == nil {
		// dummy endpoint for unregistered nodes, e.g. sandboxes or rpcs
		activeEndpoint = &pages.Endpoint{Endpoint: activeEndpointURL}
	}

	// Attach comet address to as many endpoints as possible
	cometValidators, err := cs.db.GetAllRegisteredNodes(ctx)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, fmt.Errorf("error fetching cometValidators from db: %v", err)
	}
	for _, cv := range cometValidators {
		if pep, ok := endpointMap[cv.Endpoint]; ok {
			pep.CometAddress = cv.CometAddress
		} else {
			// Rare case when node is a comet validator but not registered on eth
			endpointMap[cv.Endpoint] = &pages.Endpoint{
				Endpoint:     cv.Endpoint,
				EthAddress:   cv.EthAddress,
				CometAddress: cv.CometAddress,
			}
		}
	}

	endpoints = make([]*pages.Endpoint, 0, len(endpointMap))
	for _, ep := range endpointMap {
		endpoints = append(endpoints, ep)
	}

	return endpoints, activeEndpoint, nil
}

func (cs *Console) populateSlaReportsForEndpoints(ctx context.Context, endpoints []*pages.Endpoint, activeEndpoint *pages.Endpoint, rollupBlockEnd string) error {
	activeRollup, err := cs.getActiveSlaRollup(ctx, rollupBlockEnd)
	if err != nil {
		cs.logger.Error("failed to fetch active sla rollup", "error", err)
		return err
	}

	// Get SLA reports for active endpoint
	dbReports, err := cs.db.GetRollupReportsForNodeInTimeRange(
		ctx,
		db.GetRollupReportsForNodeInTimeRangeParams{
			Address: activeEndpoint.CometAddress, // Does not matter if unset
			Time:    cs.db.ToPgxTimestamp(activeRollup.Time.Time.Add(-24 * time.Hour)),
			Time_2:  cs.db.ToPgxTimestamp(activeRollup.Time.Time.Add(24 * time.Hour)),
		},
	)
	if err != nil {
		cs.logger.Error("failed to get rollup reports for active endpoint", "error", err)
		return err
	}

	// Attach each report to the active endpoint's history
	activeEndpoint.SlaReports = make([]*pages.SlaReport, 0, len(dbReports))
	for _, dbrep := range dbReports {
		pagerep := &pages.SlaReport{
			SlaRollupId:    dbrep.ID,
			TxHash:         dbrep.TxHash,
			BlockStart:     dbrep.BlockStart,
			BlockEnd:       dbrep.BlockEnd,
			BlocksProposed: dbrep.BlocksProposed.Int32,
			Time:           dbrep.Time.Time,
			ValidatorCount: dbrep.ValidatorCount,
		}
		attachSlaReportToEndpoint(pagerep, activeEndpoint, activeRollup.ID, !dbrep.Address.Valid)
	}

	// Now fetch the SLA reports for all endpoints, along with extra reports
	// to show a quick overview of each endpoint's SLA history

	// Organize endpoints in to map before populating with SlaRollups
	cometAddressToEndpointMap := make(map[string]*pages.Endpoint, len(endpoints))
	allCometAddresses := make([]string, len(endpoints))
	for i, ep := range endpoints {
		var key string
		if ep.CometAddress != "" {
			key = ep.CometAddress
		} else {
			// dummy address allows us to assign empty SlaReport from db later
			key = fmt.Sprintf("dummy_address_%d", i)
		}
		cometAddressToEndpointMap[key] = ep
		allCometAddresses[i] = key
	}

	// Get SLA reports for all endpoints from db in a six hour time window
	allDbReports, err := cs.db.GetRollupReportsForNodesInTimeRange(
		ctx,
		db.GetRollupReportsForNodesInTimeRangeParams{
			Column1: allCometAddresses,
			Time:    cs.db.ToPgxTimestamp(activeRollup.Time.Time.Add(-6 * time.Hour)),
			Time_2:  cs.db.ToPgxTimestamp(activeRollup.Time.Time),
		},
	)
	if err != nil {
		cs.logger.Error("failed to get rollup reports for all endpoints", "error", err)
		return err
	}

	// Attach each report to the appropriate endpoint's history
	for _, dbrep := range allDbReports {
		if ep, ok := cometAddressToEndpointMap[dbrep.Address]; ok {
			if ep.Endpoint == activeEndpoint.Endpoint {
				// we already filled out the history of the active endpoint, skip
				continue
			}
			if ep.SlaReports == nil {
				// initialize SlaReports slice with some extra headroom
				ep.SlaReports = make([]*pages.SlaReport, 0, len(allDbReports)/len(allCometAddresses)+3)
			}
			pagerep := &pages.SlaReport{
				SlaRollupId:    dbrep.ID,
				TxHash:         dbrep.TxHash,
				BlockStart:     dbrep.BlockStart,
				BlockEnd:       dbrep.BlockEnd,
				BlocksProposed: dbrep.BlocksProposed.Int32,
				Time:           dbrep.Time.Time,
				ValidatorCount: dbrep.ValidatorCount,
			}
			attachSlaReportToEndpoint(pagerep, ep, activeRollup.ID, false)
		}
	}

	return nil
}

func attachSlaReportToEndpoint(slaReport *pages.SlaReport, endpoint *pages.Endpoint, activeSlaReportId int32, forceExemptStatus bool) {
	if slaReport.SlaRollupId == activeSlaReportId {
		endpoint.ActiveReport = slaReport
	}
	if forceExemptStatus || endpointIsExemptForSlaReport(endpoint, slaReport.Time) {
		slaReport.Status = pages.SlaExempt
		slaReport.Quota = 0
	} else {
		if slaReport.ValidatorCount == 0 {
			// unexpected state, but protect against divide-by-zero panic anyway
			slaReport.ValidatorCount += 1
		}
		quota := int32(slaReport.BlockEnd-slaReport.BlockStart) / int32(slaReport.ValidatorCount)
		if quota == 0 {
			// protect against divide-by-zero panic again
			quota += 1
		}
		slaReport.Quota = quota
		faultRatio := float64(slaReport.BlocksProposed) / float64(quota)
		if faultRatio < slaMeetsThreshold && faultRatio > 0 {
			slaReport.Status = pages.SlaPartial
		} else if faultRatio == 0 {
			slaReport.Status = pages.SlaDead
		} else {
			slaReport.Status = pages.SlaMet
		}
	}
	endpoint.SlaReports = append(endpoint.SlaReports, slaReport)
}
