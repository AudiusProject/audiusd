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

	totalValidators, err := cs.db.TotalValidators(ctx)
	if err != nil {
		cs.logger.Error("Falled to get total validators", "error", err)
		return err
	}

	startTime := time.Now().Add(-7 * 24 * time.Hour)
	endTime := time.Now()

	viewEndpoints := make([]*pages.Endpoint, len(endpoints))
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

		viewSlaReports := make([]*pages.AdjudicateSlaReport, len(slaRollups))
		for i, r := range slaRollups {
			reportQuota := int32(0)
			if totalValidators > 0 {
				reportQuota = int32(r.BlockEnd-r.BlockStart) / int32(totalValidators)
			}

			var status pages.SlaStatus
			if r.BlocksProposed.Int32 == 0 {
				status = pages.SlaDead
			} else if float32(r.BlocksProposed.Int32)/float32(reportQuota) < slaMetThreshold {
				status = pages.SlaPartial
			} else {
				status = pages.SlaMet
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
	view := &pages.AdjudicatePageView{
		ServiceProvider: &pages.ServiceProvider{
			Address:   serviceProviderAddress,
			Endpoints: viewEndpoints,
		},
		StartTime: startTime,
		EndTime:   endTime,
	}

	return cs.views.RenderAdjudicateView(c, view)
}
