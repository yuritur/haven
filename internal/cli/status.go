package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	havnaws "github.com/havenapp/haven/internal/aws"
	"github.com/havenapp/haven/internal/state"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "List active deployments",
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := havnaws.LoadConfig(ctx)
	if err != nil {
		return err
	}

	stateManager, err := state.NewManager(ctx, cfg)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	deployments, err := stateManager.List(ctx)
	if err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}

	if len(deployments) == 0 {
		fmt.Println("No active deployments.")
		return nil
	}

	fmt.Printf("%-20s  %-14s  %-12s  %s\n", "ID", "MODEL", "INSTANCE", "ENDPOINT")
	fmt.Printf("%-20s  %-14s  %-12s  %s\n", "--------------------", "--------------", "------------", "--------")
	for _, d := range deployments {
		fmt.Printf("%-20s  %-14s  %-12s  %s\n", d.ID, d.Model, d.InstanceType, d.Endpoint)
	}
	return nil
}
