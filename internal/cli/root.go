package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/provider"
	awsprovider "github.com/havenapp/haven/internal/provider/aws"
)

func NewRootCmd() *cobra.Command {
	var providerName string

	root := &cobra.Command{
		Use:   "haven",
		Short: "Deploy open-source LLM models to your own cloud",
		Long:  "Haven deploys LLM models to your cloud with one command.\nYour data never leaves your infrastructure.",
	}

	root.PersistentFlags().StringVar(&providerName, "provider", "aws", "Cloud provider to use (aws)")

	root.AddCommand(newDeployCmd(&providerName))
	root.AddCommand(newDestroyCmd(&providerName))
	root.AddCommand(newStatusCmd(&providerName))

	return root
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func buildProviderAndStore(ctx context.Context, name string) (provider.Provider, provider.StateStore, error) {
	switch name {
	case "aws":
		return awsprovider.New(ctx)
	default:
		return nil, nil, fmt.Errorf("unknown provider %q - available: aws", name)
	}
}
