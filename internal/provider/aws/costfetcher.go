package aws

import (
	"context"
	"strconv"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"

	"github.com/havenapp/haven/internal/provider"
)

func (p *AWSProvider) FetchActualCost(ctx context.Context, instanceID string, from, to time.Time) (*provider.ActualCost, error) {
	client := costexplorer.NewFromConfig(p.cfg)

	out, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: awssdk.String(from.Format("2006-01-02")),
			End:   awssdk.String(to.Format("2006-01-02")),
		},
		Granularity: cetypes.GranularityDaily,
		Metrics:     []string{"UnblendedCost"},
		Filter: &cetypes.Expression{
			Dimensions: &cetypes.DimensionValues{
				Key:    cetypes.DimensionResourceId,
				Values: []string{instanceID},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if out == nil || len(out.ResultsByTime) == 0 {
		return nil, nil
	}

	var total float64
	var currency string

	for _, r := range out.ResultsByTime {
		m, ok := r.Total["UnblendedCost"]
		if !ok {
			continue
		}
		if m.Amount != nil {
			v, err := strconv.ParseFloat(*m.Amount, 64)
			if err != nil {
				continue
			}
			total += v
		}
		if m.Unit != nil && *m.Unit != "" {
			currency = *m.Unit
		}
	}

	if total == 0 && currency == "" {
		return nil, nil
	}

	return &provider.ActualCost{
		Total:    total,
		Currency: currency,
	}, nil
}
