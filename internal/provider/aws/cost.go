package aws

import (
	"context"
	"time"

	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/aws/pricing"
)

var _ provider.CostEstimator = (*AWSProvider)(nil)

func (p *AWSProvider) EstimateCost(_ context.Context, d provider.Deployment) (*provider.CostEstimate, error) {
	ebsGB := 30
	rt := models.RuntimeOllama
	if d.Runtime != "" {
		rt = models.Runtime(d.Runtime)
	}
	if spec, err := ResolveInstance(d.Model, rt); err == nil {
		ebsGB = spec.EBSVolumeGB
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

func (p *AWSProvider) ProjectCost(_ context.Context, d provider.Deployment) (*provider.CostEstimate, error) {
	ebsGB := 30
	rt := models.RuntimeOllama
	if d.Runtime != "" {
		rt = models.Runtime(d.Runtime)
	}
	if spec, err := ResolveInstance(d.Model, rt); err == nil {
		ebsGB = spec.EBSVolumeGB
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
