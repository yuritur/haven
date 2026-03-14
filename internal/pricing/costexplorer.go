package pricing

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

type ActualCost struct {
	Total       float64
	Currency    string
	LastUpdated time.Time
}

func FetchActualCost(ctx context.Context, cfg aws.Config, instanceID string, from time.Time, to time.Time) (*ActualCost, error) {
	client := costexplorer.NewFromConfig(cfg)

	out, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(from.Format("2006-01-02")),
			End:   aws.String(to.Format("2006-01-02")),
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
		return nil, nil
	}

	return parseGetCostAndUsageOutput(out)
}

func parseGetCostAndUsageOutput(out *costexplorer.GetCostAndUsageOutput) (*ActualCost, error) {
	if out == nil || len(out.ResultsByTime) == 0 {
		return nil, nil
	}

	var total float64
	var currency string
	var lastEnd time.Time

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
		if r.TimePeriod != nil && r.TimePeriod.End != nil {
			t, err := time.Parse("2006-01-02", *r.TimePeriod.End)
			if err == nil && t.After(lastEnd) {
				lastEnd = t
			}
		}
	}

	if total == 0 && currency == "" {
		return nil, nil
	}

	return &ActualCost{
		Total:       total,
		Currency:    currency,
		LastUpdated: lastEnd,
	}, nil
}
