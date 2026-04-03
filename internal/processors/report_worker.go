package processors

import (
	"context"
	"errors"
	"fmt"
	"time"

	adsapi "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	adsapimodels "github.com/LittleAksMax/amazon-ads-api-sdk-go/models"
	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/LittleAksMax/bids-util/retries"
	"github.com/google/uuid"
)

const refreshReportMaxAttempts = 3
const refreshReportBackoffTime = 2 * time.Second
const refreshPollInterval = 30 * time.Second

const cancelReportAttempts = 3
const cancelReportRetryBackoffTime = time.Second * 2

// TODO: make this adjustable? Would require changes to user service and database (and frontend)
const reportDaysLength = time.Hour * 24 * 30

type reportResult struct {
	generatedReport *adsapimodels.GeneratedReport
	err             error
}

var requestReportCfg = adsapimodels.ReportConfiguration{
	AdProduct:    adsapimodels.AdProductSP,
	GroupBy:      []adsapimodels.ReportGroupBy{adsapimodels.ReportGroupByCampaign, adsapimodels.ReportGroupByAdGroup},
	Columns:      reportColumns,
	ReportTypeID: adsapimodels.ReportTypeSponsoredProductsCampaigns,
	TimeUnit:     adsapimodels.ReportTimeUnitSummary,
	Format:       adsapimodels.ReportFormatGZIPJSON,
}

func (p *Processor) runReportWorker(ctx context.Context, cancel context.CancelCauseFunc, msg *ProcessMessage, out chan<- reportResult) {
	// Use adsClient to execute the required Ads API request for this schedule
	reportName := fmt.Sprintf("%s-%d-%s", msg.UserID.String(), msg.Profile.ProfileID, msg.DueAt.String())
	reportOpts := adsapimodels.RequestReportOptions{
		Name:          reportName,
		StartDate:     adsapimodels.FormatDate(time.Now().Add(-14 * 24 * time.Hour)),
		EndDate:       adsapimodels.FormatDate(time.Now()),
		Configuration: requestReportCfg,
	}

	// Request report generation
	p.logger.Infof("[UserID: %s; ProfileID: %d] Requesting report", msg.UserID.String(), msg.Profile.ProfileID)
	report, err := p.adsClient.ReportsService.RequestReport(ctx, msg.Profile.ProfileID, &reportOpts)
	if err != nil {
		p.logger.Errorf("[UserID: %s; ProfileID: %d] Failed to request report: %v", msg.UserID.String(), msg.Profile.ProfileID, err)
		_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, "Failed to request report from Amazon")

		// Result is the error
		out <- reportResult{
			generatedReport: nil,
			err:             err,
		}
		return
	}

	generatedReport, err := p.handleReport(ctx, msg.UserID, &msg.Profile, report)
	if err != nil {
		cancel(fmt.Errorf("report generation failed: %v", err))
	}

	out <- reportResult{
		generatedReport: generatedReport,
		err:             err,
	}
}

func (p *Processor) handleReport(ctx context.Context, userID uuid.UUID, profile *services.RegionProfile, report *adsapi.Report) (*adsapimodels.GeneratedReport, error) {
	var details *adsapimodels.ReportDetails
	var err error = nil
	for {
		err = retries.Retry(refreshReportMaxAttempts, refreshReportBackoffTime, p.logger, func(ctx context.Context) error {
			details, err = report.Refresh(ctx)
			return err
		})(ctx)
		if err != nil {
			p.logger.Errorf("[UserID: %s; ProfileID: %d; ReportID: %s] Failed to refresh report after retries: %v", userID.String(), profile.ProfileID, report.ReportID(), err)
			cancelErr := p.cancelReport(ctx, profile.ProfileID, report.ReportID())
			if cancelErr != nil {
				_, _ = p.userService.Log(ctx, userID, profile.ProfileID, "Failed to cancel report")
				p.logger.Errorf("[UserID: %s; ProfileID: %d; ReportID: %s] Failed to cancel report: %v", userID.String(), profile.ProfileID, report.ReportID(), cancelErr)
			}
			return nil, errors.Join(err, cancelErr)
		}

		p.logger.Infof("[UserID: %s; ProfileID: %d; ReportID: %s] Report status: %s", userID.String(), profile.ProfileID, report.ReportID(), details.Status)

		if details.IsTerminal() {
			break
		}

		// Sleep is cancellation-aware so profile prep errors can stop report polling quickly
		timer := time.NewTimer(refreshPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()

			// We were cancelled by another goroutine, so we must cancel the report generation
			p.logger.Infof("[UserID: %s; ProfileID: %d; ReportID: %s] Context canceled, trying to cancel report generation", userID.String(), profile.ProfileID, report.ReportID())
			err = p.cancelReport(ctx, profile.ProfileID, report.ReportID())
			if err != nil {
				_, _ = p.userService.Log(ctx, userID, profile.ProfileID, "Failed to cancel report")
				p.logger.Errorf("[UserID: %s; ProfileID: %d; ReportID: %s] Failed to cancel report: %v", userID.String(), profile.ProfileID, report.ReportID(), err)
			}
			return nil, errors.Join(ctx.Err(), err)
		case <-timer.C:
		}
	}

	// Residual error must be returned
	if err != nil {
		return nil, err
	}

	if err != nil || details.IsFailed() {
		return nil, fmt.Errorf("failed to generate report")
	}

	_, _ = p.userService.Log(ctx, userID, profile.ProfileID, "Generated report successfully")

	return report.GeneratedReport(ctx)
}

func (p *Processor) cancelReport(ctx context.Context, profileID int64, reportID string) error {
	return retries.Retry(cancelReportAttempts, cancelReportRetryBackoffTime, p.logger, func(ctx context.Context) error {
		return p.adsClient.ReportsService.CancelReport(ctx, profileID, reportID)
	})(ctx)
}
