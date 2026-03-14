package pricing

import (
	"fmt"
	"math"
	"time"
)

type CostBreakdown struct {
	EC2    float64
	EBS    float64
	EIP    float64
	Total  float64
	Uptime time.Duration
}

var ec2Prices = map[string]float64{
	"t3.large":     0.0832,
	"t3.xlarge":    0.1664,
	"g4dn.xlarge":  0.526,
	"g4dn.2xlarge": 0.752,
	"g5.xlarge":    1.006,
	"g5.2xlarge":   1.212,
	"g5.12xlarge":  5.672,
}

const (
	ebsPerGBMonth = 0.08
	avgHoursMonth = 730.0
)

func RunningHours(createdAt, now time.Time, accumulatedStopHours float64, stoppedAt *time.Time) float64 {
	total := now.Sub(createdAt).Hours()
	stopped := accumulatedStopHours
	if stoppedAt != nil {
		stopped += now.Sub(*stoppedAt).Hours()
	}
	running := total - stopped
	if running < 0 {
		return 0
	}
	return running
}

func Calculate(instanceType string, ebsGB int, runningHours float64) (CostBreakdown, error) {
	hourly, ok := ec2Prices[instanceType]
	if !ok {
		return CostBreakdown{}, fmt.Errorf("unknown instance type %q — cost estimate unavailable", instanceType)
	}

	ec2 := hourly * runningHours
	ebs := ebsPerGBMonth * float64(ebsGB) * (runningHours / avgHoursMonth)

	return CostBreakdown{
		EC2:    ec2,
		EBS:    ebs,
		EIP:    0,
		Total:  ec2 + ebs,
		Uptime: time.Duration(runningHours * float64(time.Hour)),
	}, nil
}

func Projected(instanceType string, ebsGB int, createdAt, now time.Time, accumulatedStopHours float64, stoppedAt *time.Time) (CostBreakdown, error) {
	hourly, ok := ec2Prices[instanceType]
	if !ok {
		return CostBreakdown{}, fmt.Errorf("unknown instance type %q — cost estimate unavailable", instanceType)
	}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	monthHours := monthEnd.Sub(monthStart).Hours()

	// Use month start as effective start if deployment predates current month
	effectiveStart := createdAt
	if createdAt.Before(monthStart) {
		effectiveStart = monthStart
	}

	// Running hours so far within the current month
	running := RunningHours(effectiveStart, now, accumulatedStopHours, stoppedAt)

	var projectedEC2Hours float64
	if stoppedAt != nil {
		// Currently stopped: only count hours already run, no future projection
		projectedEC2Hours = running
	} else {
		// Currently running: assume it continues until end of month
		remainingHours := monthEnd.Sub(now).Hours()
		projectedEC2Hours = running + remainingHours
	}

	ec2 := hourly * projectedEC2Hours
	// EBS is charged for the full month regardless of instance state
	ebs := ebsPerGBMonth * float64(ebsGB) * (monthHours / avgHoursMonth)

	return CostBreakdown{
		EC2:   ec2,
		EBS:   ebs,
		Total: ec2 + ebs,
	}, nil
}

func FormatUSD(amount float64) string {
	if amount > 0 && amount < 0.01 {
		return "$0.01"
	}
	return fmt.Sprintf("$%.2f", math.Round(amount*100)/100)
}

func FormatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
