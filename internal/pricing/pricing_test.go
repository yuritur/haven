package pricing

import (
	"math"
	"testing"
	"time"
)

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestCalculate(t *testing.T) {
	tests := []struct {
		name         string
		instanceType string
		ebsGB        int
		runningHours float64
		wantEC2      float64
		wantEBS      float64
		wantErr      bool
	}{
		{
			name:         "t3.large 24 hours",
			instanceType: "t3.large",
			ebsGB:        30,
			runningHours: 24,
			wantEC2:      0.0832 * 24,
			wantEBS:      0.08 * 30 * (24.0 / 730.0),
		},
		{
			name:         "g5.xlarge 48 hours",
			instanceType: "g5.xlarge",
			ebsGB:        100,
			runningHours: 48,
			wantEC2:      1.006 * 48,
			wantEBS:      0.08 * 100 * (48.0 / 730.0),
		},
		{
			name:         "unknown instance type",
			instanceType: "m5.xlarge",
			ebsGB:        30,
			runningHours: 24,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.instanceType, tt.ebsGB, tt.runningHours)
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
			if !approxEqual(got.Total, tt.wantEC2+tt.wantEBS, 0.001) {
				t.Errorf("Total = %f, want %f", got.Total, tt.wantEC2+tt.wantEBS)
			}
			if got.EIP != 0 {
				t.Errorf("EIP = %f, want 0", got.EIP)
			}
			wantUptime := time.Duration(tt.runningHours * float64(time.Hour))
			if got.Uptime != wantUptime {
				t.Errorf("Uptime = %v, want %v", got.Uptime, wantUptime)
			}
		})
	}
}

func TestRunningHours(t *testing.T) {
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
			got := RunningHours(tt.createdAt, tt.now, tt.accumulatedStopHours, tt.stoppedAt)
			if !approxEqual(got, tt.want, 0.001) {
				t.Errorf("RunningHours = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestProjected(t *testing.T) {
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
				// 5 days running so far (120h) + remaining to end of March (17 days = 408h)
				// Mar 15 00:00 to Apr 1 00:00 = 17 days
				want := 0.0832 * (120 + 408)
				return approxEqual(v, want, 0.01)
			},
			checkEBS: func(v float64) bool {
				// Full month of March = 31 days = 744 hours
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
				// RunningHours: total=5*24=120, stopped=now-stoppedAt=2*24=48, running=72h
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
				// Running since Feb 20, now Mar 10 = 18 days = 432h total running
				// Remaining: Mar 10 to Apr 1 = 22 days = 528h
				// Projected EC2 hours: 432 + 528 = 960
				want := 0.1664 * 960
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
			got, err := Projected(tt.instanceType, tt.ebsGB, tt.createdAt, tt.now, tt.accumulatedStopHours, tt.stoppedAt)
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
			if !approxEqual(got.Total, got.EC2+got.EBS, 0.001) {
				t.Errorf("Total = %f, want EC2+EBS = %f", got.Total, got.EC2+got.EBS)
			}
		})
	}
}

func TestFormatUSD(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   string
	}{
		{"normal amount", 1.234, "$1.23"},
		{"larger amount", 42.50, "$42.50"},
		{"tiny amount", 0.005, "$0.01"},
		{"zero", 0.0, "$0.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatUSD(tt.amount)
			if got != tt.want {
				t.Errorf("FormatUSD(%f) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{
			"days hours minutes",
			3*24*time.Hour + 14*time.Hour + 22*time.Minute,
			"3d 14h 22m",
		},
		{
			"hours and minutes only",
			14*time.Hour + 22*time.Minute,
			"14h 22m",
		},
		{
			"minutes only",
			22 * time.Minute,
			"22m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.d)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
