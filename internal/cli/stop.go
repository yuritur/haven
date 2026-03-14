package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/provider"
)

func newStopCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "stop <deployment-id>",
		Short: "Stop a running deployment",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("deployment ID is required\n\nUsage: haven stop <deployment-id>")
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			prov, err := buildProvider(cmd.Context(), *providerName, io.Discard)
			if err != nil {
				return err
			}
			return runStop(cmd.Context(), prov, args[0])
		},
	}
}

func runStop(ctx context.Context, prov provider.Provider, id string) error {
	d, err := prov.LoadDeployment(ctx, id)
	if err != nil {
		return fmt.Errorf("load deployment: %w", err)
	}

	if d.StoppedAt != nil {
		return fmt.Errorf("deployment already stopped")
	}

	if err := prov.Stop(ctx, d.InstanceID); err != nil {
		return fmt.Errorf("stop instance: %w", err)
	}

	now := time.Now()
	d.StoppedAt = &now

	if err := prov.SaveDeployment(ctx, *d); err != nil {
		return fmt.Errorf("save deployment: %w", err)
	}

	fmt.Printf("Deployment %s stopped.\n", id)
	return nil
}
