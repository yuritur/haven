package pricing

import (
	"math"
	"testing"
	"time"

	"github.com/havenapp/haven/internal/models"
)

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestCalcCurrent(t *testing.T) {
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name                 string
		instanceType         string
		ebsGB                int
		createdAt            time.Time
		now                  time.Time
		accumulatedStopHours float64
		stoppedAt            *time.Time
		wantEC2              float64
		wantEBS              float64
		wantEIP              float64
		wantErr              bool
	}{
		{
			name:         "t3.large 24 hours running",
			instanceType: "t3.large",
			ebsGB:        30,
			createdAt:    base,
			now:          base.Add(24 * time.Hour),
			wantEC2:      0.0832 * 24,
			wantEBS:      0.08 * 30 * (24.0 / 730.0),
			wantEIP:      0.005 * 24,
		},
		{
			name:                 "g5.xlarge 48 hours with 12 hours stopped",
			instanceType:         "g5.xlarge",
			ebsGB:                100,
			createdAt:            base,
			now:                  base.Add(48 * time.Hour),
			accumulatedStopHours: 12,
			wantEC2:              1.006 * 36,
			wantEBS:              0.08 * 100 * (48.0 / 730.0),
			wantEIP:              0.005 * 48,
		},
		{
			name:         "unknown instance type",
			instanceType: "m5.xlarge",
			ebsGB:        30,
			createdAt:    base,
			now:          base.Add(24 * time.Hour),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalcCurrent(tt.instanceType, tt.ebsGB, tt.createdAt, tt.now, tt.accumulatedStopHours, tt.stoppedAt)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !approxEqual(got.EC2, tt.wantEC2, 0.001) {
				t.Errorf("EC2 = %f, want %f", got.EC2, tt.wantEC2)
			}
			if !approxEqual(got.EBS, tt.wantEBS, 0.001) {
				t.Errorf("EBS = %f, want %f", got.EBS, tt.wantEBS)
			}
			if !approxEqual(got.EIP, tt.wantEIP, 0.001) {
				t.Errorf("EIP = %f, want %f", got.EIP, tt.wantEIP)
			}
			wantTotal := tt.wantEC2 + tt.wantEBS + tt.wantEIP
			if !approxEqual(got.Total, wantTotal, 0.001) {
				t.Errorf("Total = %f, want %f", got.Total, wantTotal)
			}
			runningHours := tt.now.Sub(tt.createdAt).Hours() - tt.accumulatedStopHours
			wantUptime := time.Duration(runningHours * float64(time.Hour))
			if got.Uptime != wantUptime {
				t.Errorf("Uptime = %v, want %v", got.Uptime, wantUptime)
			}
		})
	}
}

func TestCalcRunningHours(t *testing.T) {
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name                 string
		createdAt            time.Time
		now                  time.Time
		accumulatedStopHours float64
		stoppedAt            *time.Time
		want                 float64
	}{
		{
			name:      "no stops",
			createdAt: base,
			now:       base.Add(48 * time.Hour),
			want:      48,
		},
		{
			name:      "currently stopped",
			createdAt: base,
			now:       base.Add(48 * time.Hour),
			stoppedAt: timePtr(base.Add(24 * time.Hour)),
			want:      24,
		},
		{
			name:                 "accumulated stops",
			createdAt:            base,
			now:                  base.Add(48 * time.Hour),
			accumulatedStopHours: 10,
			want:                 38,
		},
		{
			name:                 "accumulated and currently stopped",
			createdAt:            base,
			now:                  base.Add(72 * time.Hour),
			accumulatedStopHours: 12,
			stoppedAt:            timePtr(base.Add(48 * time.Hour)),
			want:                 36, // 72 - 12 - 24
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcRunningHours(tt.createdAt, tt.now, tt.accumulatedStopHours, tt.stoppedAt)
			if !approxEqual(got, tt.want, 0.001) {
				t.Errorf("RunningHours = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestCalcProjected(t *testing.T) {
	tests := []struct {
		name                 string
		instanceType         string
		ebsGB                int
		createdAt            time.Time
		now                  time.Time
		accumulatedStopHours float64
		stoppedAt            *time.Time
		checkEC2             func(float64) bool
		checkEBS             func(float64) bool
		wantErr              bool
	}{
		{
			name:         "mid-month running",
			instanceType: "t3.large",
			ebsGB:        30,
			createdAt:    time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
			now:          time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
			checkEC2: func(v float64) bool {
				want := 0.0832 * (120 + 408)
				return approxEqual(v, want, 0.01)
			},
			checkEBS: func(v float64) bool {
				want := 0.08 * 30 * (744.0 / 730.0)
				return approxEqual(v, want, 0.01)
			},
		},
		{
			name:         "mid-month stopped",
			instanceType: "t3.large",
			ebsGB:        30,
			createdAt:    time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
			now:          time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
			stoppedAt:    timePtr(time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)),
			checkEC2: func(v float64) bool {
				want := 0.0832 * 72
				return approxEqual(v, want, 0.01)
			},
			checkEBS: func(v float64) bool {
				want := 0.08 * 30 * (744.0 / 730.0)
				return approxEqual(v, want, 0.01)
			},
		},
		{
			name:         "deployment started previous month",
			instanceType: "t3.xlarge",
			ebsGB:        30,
			createdAt:    time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
			now:          time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
			checkEC2: func(v float64) bool {
				want := 0.1664 * 744
				return approxEqual(v, want, 0.1)
			},
			checkEBS: func(v float64) bool {
				want := 0.08 * 30 * (744.0 / 730.0)
				return approxEqual(v, want, 0.01)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalcProjected(tt.instanceType, tt.ebsGB, tt.createdAt, tt.now, tt.accumulatedStopHours, tt.stoppedAt)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.checkEC2(got.EC2) {
				t.Errorf("EC2 = %f, check failed", got.EC2)
			}
			if !tt.checkEBS(got.EBS) {
				t.Errorf("EBS = %f, check failed", got.EBS)
			}
			if !approxEqual(got.Total, got.EC2+got.EBS+got.EIP, 0.001) {
				t.Errorf("Total = %f, want EC2+EBS+EIP = %f", got.Total, got.EC2+got.EBS+got.EIP)
			}
		})
	}
}

func TestCalcRunningHoursNegativeClamping(t *testing.T) {
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	got := CalcRunningHours(base, base.Add(10*time.Hour), 20, nil)
	if got != 0 {
		t.Errorf("RunningHours = %f, want 0 (accumulated stop hours exceed total)", got)
	}
}

func TestAllRegisteredInstanceTypesHavePrices(t *testing.T) {
	for _, cfg := range models.List() {
		if _, ok := ec2Prices[cfg.InstanceType]; !ok {
			t.Errorf("model registry uses %q but pricing has no rate", cfg.InstanceType)
		}
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
