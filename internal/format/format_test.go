package format

import (
	"testing"
	"time"
)

func TestUSD(t *testing.T) {
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
			got := USD(tt.amount)
			if got != tt.want {
				t.Errorf("USD(%f) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

func TestDuration(t *testing.T) {
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
			got := Duration(tt.d)
			if got != tt.want {
				t.Errorf("Duration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}
