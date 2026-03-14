package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/format"
	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/tui"
)

func newStatusCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "List active deployments",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			spinner := tui.StartSpinner("Loading deployments...")
			defer spinner.Stop()
			prov, err := buildProvider(cmd.Context(), *providerName, io.Discard)
			if err != nil {
				return err
			}
			return runStatus(cmd.Context(), prov, spinner)
		},
	}
}

func runStatus(ctx context.Context, prov provider.Provider, spinner *tui.Spinner) error {
	deployments, err := prov.List(ctx)
	if err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}

	if spinner != nil {
		spinner.Stop()
	}

	if len(deployments) == 0 {
		fmt.Println("No active deployments.")
		return nil
	}

	fmt.Printf("\033[33m%-20s  %-6s  %-14s  %-12s  %-9s  %-10s  %s\033[0m\n", "ID", "CLOUD", "MODEL", "INSTANCE", "STATE", "EST.COST", "ENDPOINT")
	fmt.Printf("\033[33m%-20s  %-6s  %-14s  %-12s  %-9s  %-10s  %s\033[0m\n", "--------------------", "------", "--------------", "------------", "---------", "----------", "--------")

	ce, hasCost := prov.(provider.CostEstimator)
	for _, d := range deployments {
		state := "running"
		if d.StoppedAt != nil {
			state = "stopped"
		}

		costStr := "N/A"
		if hasCost {
			if est, err := ce.EstimateCost(ctx, d); err == nil {
				costStr = format.USD(est.Total)
			}
		}

		fmt.Printf("%-20s  %-6s  %-14s  %-12s  %-9s  %-10s  %s\n", d.ID, d.Provider, d.Model, d.InstanceType, state, costStr, d.Endpoint)
	}
	return nil
}
