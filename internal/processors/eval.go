package processors

import (
	"context"

	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/LittleAksMax/bidscript"
	"github.com/LittleAksMax/bidscript/evaluator"
)

func (p *Processor) evaluate(ctx context.Context, policy services.Policy, report ReportColumns) (evaluator.Result, error) {
	var acos float64
	if report.Sales1d != 0.0 {
		acos = report.Cost / report.Sales1d
	} else {
		acos = -1
	}
	roas := 1.0 / acos

	metrics := evaluator.Metrics{
		Impressions: report.Impressions,
		Spend:       report.Cost,
		Sales:       report.Sales1d,
		Clicks:      report.Clicks,
		CTR:         report.ClickThroughRate,
		CPC:         report.CostPerClick,
		Orders:      report.Purchases1d,
		ACoS:        acos,
		RoaS:        roas,
	}
	program := bidscript.Program{
		Source: policy.Script,
	}
	return bidscript.Eval(ctx, program, &metrics)
}
