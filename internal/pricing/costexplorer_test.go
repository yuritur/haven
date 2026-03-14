package pricing

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

func TestParseGetCostAndUsageOutput(t *testing.T) {
	tests := []struct {
		name         string
		out          *costexplorer.GetCostAndUsageOutput
		wantNil      bool
		wantTotal    float64
		wantCurrency string
		wantLastEnd  string
	}{
		{
			name:    "nil output",
			out:     nil,
			wantNil: true,
		},
		{
			name:    "empty results",
			out:     &costexplorer.GetCostAndUsageOutput{},
			wantNil: true,
		},
		{
			name: "single day",
			out: &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []cetypes.ResultByTime{
					{
						TimePeriod: &cetypes.DateInterval{
							Start: aws.String("2026-03-10"),
							End:   aws.String("2026-03-11"),
						},
						Total: map[string]cetypes.MetricValue{
							"UnblendedCost": {
								Amount: aws.String("1.50"),
								Unit:   aws.String("USD"),
							},
						},
					},
				},
			},
			wantTotal:    1.50,
			wantCurrency: "USD",
			wantLastEnd:  "2026-03-11",
		},
		{
			name: "multiple days summed",
			out: &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []cetypes.ResultByTime{
					{
						TimePeriod: &cetypes.DateInterval{
							Start: aws.String("2026-03-10"),
							End:   aws.String("2026-03-11"),
						},
						Total: map[string]cetypes.MetricValue{
							"UnblendedCost": {
								Amount: aws.String("1.25"),
								Unit:   aws.String("USD"),
							},
						},
					},
					{
						TimePeriod: &cetypes.DateInterval{
							Start: aws.String("2026-03-11"),
							End:   aws.String("2026-03-12"),
						},
						Total: map[string]cetypes.MetricValue{
							"UnblendedCost": {
								Amount: aws.String("2.75"),
								Unit:   aws.String("USD"),
							},
						},
					},
					{
						TimePeriod: &cetypes.DateInterval{
							Start: aws.String("2026-03-12"),
							End:   aws.String("2026-03-13"),
						},
						Total: map[string]cetypes.MetricValue{
							"UnblendedCost": {
								Amount: aws.String("0.50"),
								Unit:   aws.String("USD"),
							},
						},
					},
				},
			},
			wantTotal:    4.50,
			wantCurrency: "USD",
			wantLastEnd:  "2026-03-13",
		},
		{
			name: "zero cost results return nil",
			out: &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []cetypes.ResultByTime{
					{
						TimePeriod: &cetypes.DateInterval{
							Start: aws.String("2026-03-10"),
							End:   aws.String("2026-03-11"),
						},
						Total: map[string]cetypes.MetricValue{
							"UnblendedCost": {
								Amount: aws.String("0"),
								Unit:   aws.String(""),
							},
						},
					},
				},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGetCostAndUsageOutput(tt.out)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if !approxEqual(got.Total, tt.wantTotal, 0.001) {
				t.Errorf("Total = %f, want %f", got.Total, tt.wantTotal)
			}
			if got.Currency != tt.wantCurrency {
				t.Errorf("Currency = %q, want %q", got.Currency, tt.wantCurrency)
			}
			if tt.wantLastEnd != "" {
				gotEnd := got.LastUpdated.Format("2006-01-02")
				if gotEnd != tt.wantLastEnd {
					t.Errorf("LastUpdated = %q, want %q", gotEnd, tt.wantLastEnd)
				}
			}
		})
	}
}
