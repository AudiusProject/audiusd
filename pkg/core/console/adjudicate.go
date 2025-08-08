package console

import (
	"errors"
	"time"

	"connectrpc.com/connect"
	ethv1 "github.com/AudiusProject/audiusd/pkg/api/eth/v1"
	"github.com/AudiusProject/audiusd/pkg/core/console/views/pages"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

const slaMetThreshold float32 = 0.5

func (cs *Console) adjudicateFragment(c echo.Context) error {
	ctx := c.Request().Context()

	serviceProviderAddress := c.Param("sp")
	_, err := cs.eth.GetServiceProvider(
		ctx,
		connect.NewRequest(&ethv1.GetServiceProviderRequest{Address: serviceProviderAddress}),
	)
	if err != nil {
		cs.logger.Error("Falled to get service provider", "address", serviceProviderAddress, "error", err)
		return err
	}

	endpointsResp, err := cs.eth.GetRegisteredEndpointsForServiceProvider(
		ctx,
		connect.NewRequest(&ethv1.GetRegisteredEndpointsForServiceProviderRequest{Owner: serviceProviderAddress}),
	)
	if err != nil {
		cs.logger.Error("Falled to get service provider endpoints", "address", serviceProviderAddress, "error", err)
		return err
	}
	endpoints := endpointsResp.Msg.Endpoints

	// Get total number of cometbft validators in order to calculate SLA performance later
	totalValidators, err := cs.db.TotalValidators(ctx)
	if err != nil {
		cs.logger.Error("Falled to get total validators", "error", err)
		return err
	}

	// configure start and end times
	startTime := time.Now().Add(-7 * 24 * time.Hour)
	endTime := time.Now()
	if c.QueryParam("start") != "" {
		if parsed, err := time.Parse("2006-01-02T15:04", c.QueryParam("start")); err != nil {
			cs.logger.Warn("failed to parse start time from query string", "error", err)
		} else {
			startTime = parsed
		}
	}
	if c.QueryParam("end") != "" {
		if parsed, err := time.Parse("2006-01-02T15:04", c.QueryParam("end")); err != nil {
			cs.logger.Warn("failed to parse end time from query string", "error", err)
		} else {
			endTime = parsed
		}
	}

	// Populate endpoints and their SLAs for the view model
	viewEndpoints := make([]*pages.Endpoint, len(endpoints))
	totalMetSlas, totalPartialSlas, totalDeadSlas := 0, 0, 0
	for i, ep := range endpoints {
		viewEndpoints[i] = &pages.Endpoint{
			Endpoint:   ep.Endpoint,
			EthAddress: ep.DelegateWallet,
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			cs.logger.Error("Falled to get rollups in time range", "start time", startTime, "end time", endTime, "error", err)
			return err
		}

		var cometAddress string
		validator, err := cs.db.GetNodeByEndpoint(ctx, ep.Endpoint)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			cs.logger.Error("Falled to get cometbft validator for endpoint", "endpoint", ep.Endpoint, "error", err)
			return err
		} else if !errors.Is(err, pgx.ErrNoRows) {
			cometAddress = validator.CometAddress
		}
		viewEndpoints[i].CometAddress = cometAddress

		slaRollups, err := cs.db.GetRollupReportsForNodeInTimeRange(
			ctx,
			db.GetRollupReportsForNodeInTimeRangeParams{
				Address: cometAddress,
				Time:    cs.db.ToPgxTimestamp(startTime),
				Time_2:  cs.db.ToPgxTimestamp(endTime),
			},
		)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			cs.logger.Error("Failed to get rollups from db for node", "address", cometAddress, "start time", startTime, "end time", endTime, "error", err)
			return err
		}

		// Calculate the status of each SLA rollup as met, partially met, or dead (0% met)
		viewSlaReports := make([]*pages.AdjudicateSlaReport, len(slaRollups))
		for i, r := range slaRollups {
			reportQuota := int32(0)
			if totalValidators > 0 {
				reportQuota = int32(r.BlockEnd-r.BlockStart) / int32(totalValidators)
			}

			var status pages.SlaStatus
			if r.BlocksProposed.Int32 == 0 {
				status = pages.SlaDead
				totalDeadSlas += 1
			} else if float32(r.BlocksProposed.Int32)/float32(reportQuota) < slaMetThreshold {
				status = pages.SlaPartial
				totalPartialSlas += 1
			} else {
				status = pages.SlaMet
				totalMetSlas += 1
			}

			viewSlaReports[i] = &pages.AdjudicateSlaReport{
				TxHash:     r.TxHash,
				Status:     status,
				BlockStart: r.BlockStart,
				BlockEnd:   r.BlockEnd,
				Time:       r.Time.Time,
			}
		}

		viewEndpoints[i].SlaReports = viewSlaReports
	}

	// Calculate how much this service provider earns in rewards per round
	// and how much should be slashed based on the SLA performance across all owned endpoints
	stakingResp, err := cs.eth.GetStakingMetadataForServiceProvider(
		ctx,
		connect.NewRequest(&ethv1.GetStakingMetadataForServiceProviderRequest{Address: serviceProviderAddress}),
	)
	if err != nil {
		cs.logger.Error("Falled to get service provider staking metadata", "address", serviceProviderAddress, "error", err)
		return err
	}
	periodDays := int64(endTime.Sub(startTime).Hours() / 24)
	cs.logger.Infof("**** DELETEMEcs period days: %d", periodDays)
	totalRewards := int64(stakingResp.Msg.RewardsPerRound * periodDays / 7)
	cs.logger.Infof("**** DELETEMEcs rewards per round: %d", stakingResp.Msg.RewardsPerRound)
	cs.logger.Infof("**** DELETEMEcs totalRewards: %d", totalRewards)
	totalUnearnedRewards := int64(float64(totalDeadSlas) / float64(totalMetSlas+totalPartialSlas+totalDeadSlas) * float64(totalRewards))

	view := &pages.AdjudicatePageView{
		ServiceProvider: &pages.ServiceProvider{
			Address:   serviceProviderAddress,
			Endpoints: viewEndpoints,
		},
		StartTime:              startTime,
		EndTime:                endTime,
		MetSlas:                totalMetSlas,
		PartialSlas:            totalPartialSlas,
		DeadSlas:               totalDeadSlas,
		TotalStaked:            stakingResp.Msg.TotalStaked,
		TotalSPRewards:         totalRewards,
		TotalUnearnedSPRewards: totalUnearnedRewards,
	}

	return cs.views.RenderAdjudicateView(c, view)
}
