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

func newStartCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:     "start [deployment-id]",
		Short:   "Start a stopped deployment",
		Long:    "Start a stopped deployment.\nIf only one deployment exists, it is selected automatically.",
		Example: "  haven start\n  haven start haven-a1b2c3d4",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spinner := tui.StartSpinner("Starting deployment...")
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
			return runStart(cmd.Context(), prov, d.ID, spinner)
		},
	}
}

func runStart(ctx context.Context, prov provider.Provider, id string, spinner *tui.Spinner) error {
	d, err := prov.LoadDeployment(ctx, id)
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

	if err := prov.SaveDeployment(ctx, *d); err != nil {
		return fmt.Errorf("save deployment: %w", err)
	}

	if spinner != nil {
		spinner.Stop()
	}
	fmt.Printf("Deployment \033[33m%s\033[0m started.\n", id)
	return nil
}
