package format

import (
	"fmt"
	"math"
	"time"
)

func USD(amount float64) string {
	if amount > 0 && amount < 0.01 {
		return "$0.01"
	}
	return fmt.Sprintf("$%.2f", math.Round(amount*100)/100)
}

func Duration(d time.Duration) string {
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
