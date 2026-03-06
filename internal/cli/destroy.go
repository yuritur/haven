package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	havnaws "github.com/havenapp/haven/internal/aws"
	"github.com/havenapp/haven/internal/cfn"
	"github.com/havenapp/haven/internal/state"
)

var destroyCmd = &cobra.Command{
	Use:     "destroy <deployment-id>",
	Short:   "Destroy a deployment and release all AWS resources",
	Example: "  haven destroy haven-a1b2c3d4",
	Args:    cobra.ExactArgs(1),
	RunE:    runDestroy,
}

func runDestroy(cmd *cobra.Command, args []string) error {
	deploymentID := args[0]
	ctx := context.Background()

	cfg, err := havnaws.LoadConfig(ctx)
	if err != nil {
		return err
	}

	stateManager, err := state.NewManager(ctx, cfg)
	if err != nil {
		return fmt.Errorf("load state manager: %w", err)
	}

	deployment, err := stateManager.Load(ctx, deploymentID)
	if err != nil {
		return err
	}

	fmt.Printf("Destroying %s (%s on %s)...\n\n", deployment.ID, deployment.Model, deployment.InstanceType)

	if err := cfn.Destroy(ctx, cfg, deployment.StackName); err != nil {
		return fmt.Errorf("CloudFormation destroy: %w", err)
	}

	if err := stateManager.Delete(ctx, deploymentID); err != nil {
		fmt.Printf("Warning: failed to delete state for %s: %v\n", deploymentID, err)
	}

	fmt.Printf("\nDestroyed %s. All AWS resources released.\n", deploymentID)
	return nil
}
