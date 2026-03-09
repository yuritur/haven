package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/provider"
	awsprovider "github.com/havenapp/haven/internal/provider/aws"
)

func NewRootCmd() *cobra.Command {
	var providerName string
	var verbose bool

	root := &cobra.Command{
		Use:   "haven",
		Short: "Deploy open-source LLM models to your own cloud",
		Long:  "Haven deploys LLM models to your cloud with one command.\nYour data never leaves your infrastructure.",
	}

	root.PersistentFlags().StringVar(&providerName, "provider", "aws", "Cloud provider to use (aws)")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed provider resource events")

	root.AddCommand(newDeployCmd(&providerName, &verbose))
	root.AddCommand(newDestroyCmd(&providerName, &verbose))
	root.AddCommand(newStatusCmd(&providerName))
	root.AddCommand(newCertCmd(&providerName))

	return root
}

func Execute() {
	cmd := NewRootCmd()
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[31merror: %v\033[0m\n", err)
		os.Exit(1)
	}
}

func buildProviderAndStore(ctx context.Context, name string, out io.Writer) (provider.Provider, provider.StateStore, error) {
	switch name {
	case "aws":
		return awsprovider.New(ctx, out)
	default:
		return nil, nil, fmt.Errorf("unknown provider %q - available: aws", name)
	}
}
