package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/provider"
)

func newStartCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "start <deployment-id>",
		Short: "Start a stopped deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prov, store, err := buildProvider(cmd.Context(), *providerName, io.Discard)
			if err != nil {
				return err
			}
			return runStart(cmd.Context(), prov, store, args[0])
		},
	}
}

func runStart(ctx context.Context, prov provider.Provider, store provider.StateStore, id string) error {
	d, err := store.Load(ctx, id)
	if err != nil {
		return fmt.Errorf("load deployment: %w", err)
	}

	if d.StoppedAt == nil {
		return fmt.Errorf("deployment is not stopped")
	}

	if err := prov.Start(ctx, d.InstanceID); err != nil {
		return fmt.Errorf("start instance: %w", err)
	}

	d.AccumulatedStopHours += time.Since(*d.StoppedAt).Hours()
	d.StoppedAt = nil

	if err := store.Save(ctx, *d); err != nil {
		return fmt.Errorf("save deployment: %w", err)
	}

	fmt.Printf("Deployment %s started.\n", id)
	return nil
}
