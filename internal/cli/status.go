package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/provider"
)

func newStatusCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "List active deployments",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			prompter := newTerminalPrompter()
			_, store, err := buildProvider(cmd.Context(), *providerName, prompter, io.Discard)
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

	fmt.Printf("%-20s  %-6s  %-14s  %-12s  %s\n", "ID", "CLOUD", "MODEL", "INSTANCE", "ENDPOINT")
	fmt.Printf("%-20s  %-6s  %-14s  %-12s  %s\n", "--------------------", "------", "--------------", "------------", "--------")
	for _, d := range deployments {
		fmt.Printf("%-20s  %-6s  %-14s  %-12s  %s\n", d.ID, d.Provider, d.Model, d.InstanceType, d.Endpoint)
	}
	return nil
}
