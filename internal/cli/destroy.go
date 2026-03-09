package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/tui"
)

func newDestroyCmd(providerName *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:     "destroy <deployment-id>",
		Short:   "Destroy a deployment and release all cloud resources",
		Example: "  haven destroy haven-a1b2c3d4",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out io.Writer = io.Discard
			if *verbose {
				out = os.Stdout
			}
			prov, store, err := buildProvider(cmd.Context(), *providerName, out)
			if err != nil {
				return err
			}
			return runDestroy(cmd.Context(), prov, store, args[0], *verbose)
		},
	}
}

func runDestroy(ctx context.Context, prov provider.Provider, store provider.StateStore, deploymentID string, verbose bool) error {
	deployment, err := store.Load(ctx, deploymentID)
	if err != nil {
		return err
	}

	fmt.Printf("\033[33mDestroying\033[0m %s (%s on %s)...\n\n", deployment.ID, deployment.Model, deployment.InstanceType)

	var spin *tui.Spinner
	if !verbose {
		spin = tui.StartSpinner("Tearing down resources...")
	}

	err = prov.Destroy(ctx, deployment.ProviderRef)

	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		return fmt.Errorf("destroy: %w", err)
	}

	if err := store.Delete(ctx, deploymentID); err != nil {
		fmt.Printf("Warning: failed to delete state for %s: %v\n", deploymentID, err)
	}

	fmt.Printf("\nDestroyed %s. All resources released.\n", deploymentID)
	return nil
}
