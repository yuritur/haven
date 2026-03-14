package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/pricing"
	"github.com/havenapp/haven/internal/provider"
)

func newCostCmd(providerName *string) *cobra.Command {
	var projected bool
	cmd := &cobra.Command{
		Use:     "cost [deployment-id]",
		Short:   "Show estimated cost for a deployment",
		Long:    "Show estimated cost for a deployment.\nIf only one deployment exists, it is selected automatically.",
		Example: "  haven cost\n  haven cost haven-a1b2c3d4",
		Args:    cobra.MaximumNArgs(1),
	}
	cmd.Flags().BoolVar(&projected, "projected", false, "Show projected cost to end of month")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
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
		return runCost(cmd.Context(), prov, d.ID, projected, cmd.OutOrStdout())
	}
	return cmd
}

func runCost(ctx context.Context, prov provider.Provider, id string, projected bool, w io.Writer) error {
	d, err := prov.LoadDeployment(ctx, id)
	if err != nil {
		return fmt.Errorf("load deployment: %w", err)
	}

	ebsGB := 30
	if mc, err := models.Lookup(d.Model); err == nil {
		ebsGB = mc.EBSVolumeGB
	}

	now := time.Now()
	runningHours := pricing.RunningHours(d.CreatedAt, now, d.AccumulatedStopHours, d.StoppedAt)

	y := func(s string) string { return "\033[33m" + s + "\033[0m" }

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Cost for %s (%s on %s)\n", y(d.ID), y(d.Model), y(d.InstanceType))

	totalHours := now.Sub(d.CreatedAt).Hours()
	breakdown, calcErr := pricing.Calculate(d.InstanceType, ebsGB, runningHours, totalHours)
	if calcErr != nil {
		fmt.Fprintf(&buf, "Uptime: %s\n\n", y(pricing.FormatDuration(time.Duration(runningHours*float64(time.Hour)))))
		fmt.Fprintf(&buf, "Warning: %v\n", calcErr)
	} else {
		fmt.Fprintf(&buf, "Uptime: %s\n", y(pricing.FormatDuration(breakdown.Uptime)))
		fmt.Fprintf(&buf, "\nEstimated (us-east-1 rates):\n")
		fmt.Fprintf(&buf, "  EC2 compute    %s\n", y(pricing.FormatUSD(breakdown.EC2)))
		fmt.Fprintf(&buf, "  EBS storage    %s\n", y(pricing.FormatUSD(breakdown.EBS)))
		fmt.Fprintf(&buf, "  EIP            %s\n", y(pricing.FormatUSD(breakdown.EIP)))
		fmt.Fprintf(&buf, "  %s\n", y("\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500"))
		fmt.Fprintf(&buf, "  Total          %s\n", y(pricing.FormatUSD(breakdown.Total)))
	}

	if cf, ok := prov.(provider.CostFetcher); ok {
		if actual, err := cf.FetchActualCost(ctx, d.InstanceID, d.CreatedAt, now); err == nil && actual != nil {
			fmt.Fprintf(&buf, "\nAlready counted (AWS billing, ~24h delay):\n")
			fmt.Fprintf(&buf, "  Total          %s\n", y(pricing.FormatUSD(actual.Total)))
		}
	}

	if projected && calcErr == nil {
		if proj, err := pricing.Projected(d.InstanceType, ebsGB, d.CreatedAt, now, d.AccumulatedStopHours, d.StoppedAt); err == nil {
			fmt.Fprintf(&buf, "\nProjected to end of month:  %s\n", y(pricing.FormatUSD(proj.Total)))
		}
	}

	fmt.Fprint(w, buf.String())
	return nil
}
