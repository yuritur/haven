package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newDestroyCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:     "destroy <deployment-id>",
		Short:   "Destroy a deployment and release all cloud resources",
		Example: "  haven destroy haven-a1b2c3d4",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDestroy(cmd.Context(), *providerName, args[0])
		},
	}
}

func runDestroy(ctx context.Context, providerName, deploymentID string) error {
	prov, store, err := buildProviderAndStore(ctx, providerName)
	if err != nil {
		return err
	}

	deployment, err := store.Load(ctx, deploymentID)
	if err != nil {
		return err
	}

	fmt.Printf("Destroying %s (%s on %s)...\n\n", deployment.ID, deployment.Model, deployment.InstanceType)

	if err := prov.Destroy(ctx, deployment.ProviderRef); err != nil {
		return fmt.Errorf("destroy: %w", err)
	}

	if err := store.Delete(ctx, deploymentID); err != nil {
		fmt.Printf("Warning: failed to delete state for %s: %v\n", deploymentID, err)
	}

	fmt.Printf("\nDestroyed %s. All resources released.\n", deploymentID)
	return nil
}
