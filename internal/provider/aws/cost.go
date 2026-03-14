package aws

import (
	"context"
	"time"

	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/aws/pricing"
)

var _ provider.CostEstimator = (*AWSProvider)(nil)

// EstimateCost returns the estimated cost of a deployment based on
// EC2, EBS, and EIP rates for us-east-1. On unknown instance types
// it returns a partial estimate (with Uptime set) and an error.
func (p *AWSProvider) EstimateCost(_ context.Context, d provider.Deployment) (*provider.CostEstimate, error) {
	ebsGB := 30
	if mc, err := models.Lookup(d.Model); err == nil {
		ebsGB = mc.EBSVolumeGB
	}

	now := time.Now()
	cb, err := pricing.CalcCurrent(d.InstanceType, ebsGB, d.CreatedAt, now, d.AccumulatedStopHours, d.StoppedAt)
	if err != nil {
		uptime := pricing.CalcRunningHours(d.CreatedAt, now, d.AccumulatedStopHours, d.StoppedAt)
		return &provider.CostEstimate{
			Uptime: time.Duration(uptime * float64(time.Hour)),
		}, err
	}

	return &provider.CostEstimate{
		Total:  cb.Total,
		Uptime: cb.Uptime,
	}, nil
}

// ProjectCost extrapolates the deployment cost to the end of the current
// calendar month. If the instance is stopped, only already-accrued compute
// hours are counted; EBS and EIP are projected for the full month.
func (p *AWSProvider) ProjectCost(_ context.Context, d provider.Deployment) (*provider.CostEstimate, error) {
	ebsGB := 30
	if mc, err := models.Lookup(d.Model); err == nil {
		ebsGB = mc.EBSVolumeGB
	}

	now := time.Now()
	cb, err := pricing.CalcProjected(d.InstanceType, ebsGB, d.CreatedAt, now, d.AccumulatedStopHours, d.StoppedAt)
	if err != nil {
		return nil, err
	}

	return &provider.CostEstimate{
		Total: cb.Total,
	}, nil
}
