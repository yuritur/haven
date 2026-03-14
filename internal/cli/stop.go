package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/tui"
)

func newStopCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:     "stop [deployment-id]",
		Short:   "Stop a running deployment",
		Long:    "Stop a running deployment.\nIf only one deployment exists, it is selected automatically.",
		Example: "  haven stop\n  haven stop haven-a1b2c3d4",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spinner := tui.StartSpinner("Stopping deployment...")
			defer spinner.Stop()
			prompter := newTerminalPrompter()
			prov, err := buildProvider(cmd.Context(), *providerName, io.Discard)
			if err != nil {
				return err
			}
			var id string
			if len(args) == 1 {
				id = args[0]
			}
			d, err := resolveDeployment(cmd.Context(), prov, prompter, id)
			if err != nil {
				return err
			}
			return runStop(cmd.Context(), prov, d.ID, spinner)
		},
	}
}

func runStop(ctx context.Context, prov provider.Provider, id string, spinner *tui.Spinner) error {
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

	if spinner != nil {
		spinner.Stop()
	}
	fmt.Printf("Deployment \033[33m%s\033[0m stopped.\n", id)
	return nil
}
