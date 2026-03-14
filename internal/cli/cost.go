package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/format"
	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/tui"
)

func newCostCmd(providerName *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cost [deployment-id]",
		Short:   "Show estimated cost for a deployment",
		Long:    "Show estimated cost for a deployment.\nIf only one deployment exists, it is selected automatically.",
		Example: "  haven cost\n  haven cost haven-a1b2c3d4",
		Args:    cobra.MaximumNArgs(1),
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		spinner := tui.StartSpinner("Loading cost...")
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
		return runCost(cmd.Context(), prov, d.ID, cmd.OutOrStdout(), spinner)
	}
	return cmd
}

func runCost(ctx context.Context, prov provider.Provider, id string, w io.Writer, spinner *tui.Spinner) error {
	d, err := prov.LoadDeployment(ctx, id)
	if err != nil {
		return fmt.Errorf("load deployment: %w", err)
	}

	y := func(s string) string { return "\033[33m" + s + "\033[0m" }
	o := func(s string) string { return "\033[38;5;208m" + s + "\033[0m" }

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Cost for %s (%s on %s, %s)\n", y(d.ID), y(d.Model), y(d.InstanceType), o(d.Provider))

	ce, ok := prov.(provider.CostEstimator)
	if !ok {
		fmt.Fprintf(&buf, "Cost estimation not available for this provider.\n")
		if spinner != nil {
			spinner.Stop()
		}
		fmt.Fprint(w, buf.String())
		return nil
	}

	est, estErr := ce.EstimateCost(ctx, *d)
	if estErr != nil {
		if est != nil {
			fmt.Fprintf(&buf, "Uptime:     %s\n", y(format.Duration(est.Uptime)))
		}
		fmt.Fprintf(&buf, "Warning: %v\n", estErr)
	} else {
		fmt.Fprintf(&buf, "Uptime:     %s\n", y(format.Duration(est.Uptime)))
		fmt.Fprintf(&buf, "Estimated:  %s\n", y(format.USD(est.Total)))
	}

	if estErr == nil {
		if proj, err := ce.ProjectCost(ctx, *d); err == nil {
			fmt.Fprintf(&buf, "Projected:  %s  (to end of month)\n", y(format.USD(proj.Total)))
		}
	}

	if cf, ok := prov.(provider.CostFetcher); ok {
		now := time.Now()
		actual, err := cf.FetchActualCost(ctx, d.InstanceID, d.CreatedAt, now)
		if err == nil && actual != nil {
			fmt.Fprintf(&buf, "Actual:     %s  (billing)\n", y(format.USD(actual.Total)))
		} else {
			fmt.Fprintf(&buf, "Actual:     %s\n", y("pending"))
		}
	}

	if spinner != nil {
		spinner.Stop()
	}
	fmt.Fprint(w, buf.String())
	return nil
}
