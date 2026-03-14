package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/pricing"
	"github.com/havenapp/haven/internal/provider"
)

func newStatusCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "List active deployments",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, store, err := buildProvider(cmd.Context(), *providerName, io.Discard)
			if err != nil {
				return err
			}
			return runStatus(cmd.Context(), store)
		},
	}
}

func runStatus(ctx context.Context, store provider.StateStore) error {
	deployments, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}

	if len(deployments) == 0 {
		fmt.Println("No active deployments.")
		return nil
	}

	fmt.Printf("\033[33m%-20s  %-6s  %-14s  %-12s  %-9s  %-10s  %s\033[0m\n", "ID", "CLOUD", "MODEL", "INSTANCE", "STATE", "EST.COST", "ENDPOINT")
	fmt.Printf("\033[33m%-20s  %-6s  %-14s  %-12s  %-9s  %-10s  %s\033[0m\n", "--------------------", "------", "--------------", "------------", "---------", "----------", "--------")
	now := time.Now()
	for _, d := range deployments {
		state := "running"
		if d.StoppedAt != nil {
			state = "stopped"
		}

		costStr := "N/A"
		ebsGB := 30
		if mc, err := models.Lookup(d.Model); err == nil {
			ebsGB = mc.EBSVolumeGB
		}
		runHours := pricing.RunningHours(d.CreatedAt, now, d.AccumulatedStopHours, d.StoppedAt)
		if cb, err := pricing.Calculate(d.InstanceType, ebsGB, runHours); err == nil {
			costStr = pricing.FormatUSD(cb.Total)
		}

		fmt.Printf("%-20s  %-6s  %-14s  %-12s  %-9s  %-10s  %s\n", d.ID, d.Provider, d.Model, d.InstanceType, state, costStr, d.Endpoint)
	}
	return nil
}
