package processors

import (
	"context"
	"fmt"
	"time"

	adsapi "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	adsapimodels "github.com/LittleAksMax/amazon-ads-api-sdk-go/models"
	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/google/uuid"
)

const refreshReportMaxAttempts = 3
const refreshReportTimeout = 20 * time.Second

type reportResult struct {
	generatedReport *adsapimodels.GeneratedReport
	err             error
}

func (p *Processor) runReportWorker(ctx context.Context, cancel context.CancelCauseFunc, userID uuid.UUID, profile *services.RegionProfile, report *adsapi.Report, out chan<- reportResult) {
	generatedReport, err := p.handleReport(ctx, userID, profile, report)
	if err != nil {
		cancel(fmt.Errorf("report generation failed: %w", err))
	}

	out <- reportResult{
		generatedReport: generatedReport,
		err:             err,
	}
}

func (p *Processor) handleReport(ctx context.Context, userID uuid.UUID, profile *services.RegionProfile, report *adsapi.Report) (*adsapimodels.GeneratedReport, error) {
	var details *adsapimodels.ReportDetails
	var err error = nil
	for i := 0; i < refreshReportMaxAttempts; i++ {
		details, err = report.Refresh(ctx)
		if err != nil {
			p.logger.Errorf("[User %s; Profile %d] Error refreshing report: %v", userID.String(), profile.ProfileID, err)
		} else {
			p.logger.Infof("[User %s; Profile %d] Report status: %s", userID.String(), profile.ProfileID, details.Status)

			if details.IsTerminal() {
				break
			}
		}

		// Sleep is cancellation-aware so profile prep errors can stop report polling quickly
		timer := time.NewTimer(refreshReportTimeout)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
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

	return report.GeneratedReport(ctx)
}
