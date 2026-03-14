package pricing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

type mockCEClient struct {
	out *costexplorer.GetCostAndUsageOutput
	err error
}

func (m *mockCEClient) GetCostAndUsage(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
	return m.out, m.err
}

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

func TestFetchActualCost(t *testing.T) {
	ctx := context.Background()
	from := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)

	t.Run("api error is returned", func(t *testing.T) {
		client := &mockCEClient{err: errors.New("access denied")}
		got, err := FetchActualCost(ctx, client, "i-abc123", from, to)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got != nil {
			t.Fatalf("expected nil result, got %+v", got)
		}
	})

	t.Run("successful response parsed", func(t *testing.T) {
		client := &mockCEClient{
			out: &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []cetypes.ResultByTime{
					{
						TimePeriod: &cetypes.DateInterval{
							Start: aws.String("2026-03-10"),
							End:   aws.String("2026-03-11"),
						},
						Total: map[string]cetypes.MetricValue{
							"UnblendedCost": {
								Amount: aws.String("3.00"),
								Unit:   aws.String("USD"),
							},
						},
					},
				},
			},
		}
		got, err := FetchActualCost(ctx, client, "i-abc123", from, to)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if !approxEqual(got.Total, 3.00, 0.001) {
			t.Errorf("Total = %f, want 3.00", got.Total)
		}
		if got.Currency != "USD" {
			t.Errorf("Currency = %q, want USD", got.Currency)
		}
	})

	t.Run("empty response returns nil", func(t *testing.T) {
		client := &mockCEClient{out: &costexplorer.GetCostAndUsageOutput{}}
		got, err := FetchActualCost(ctx, client, "i-abc123", from, to)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})
}
