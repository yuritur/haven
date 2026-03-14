package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/provider"
	awsprovider "github.com/havenapp/haven/internal/provider/aws"
)

const banner = "\033[33m" +
	" ██╗  ██╗ █████╗ ██╗   ██╗███████╗███╗   ██╗\n" +
	" ██║  ██║██╔══██╗██║   ██║██╔════╝████╗  ██║\n" +
	" ███████║███████║██║   ██║█████╗  ██╔██╗ ██║\n" +
	" ██╔══██║██╔══██║╚██╗ ██╔╝██╔══╝  ██║╚██╗██║\n" +
	" ██║  ██║██║  ██║ ╚████╔╝ ███████╗██║ ╚████║\n" +
	" ╚═╝  ╚═╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═══╝\033[0m"

func NewRootCmd() *cobra.Command {
	var providerName string
	var verbose bool

	root := &cobra.Command{
		Use:   "haven",
		Short: "Deploy open-source LLM models to your own cloud",
		Long: banner + "\n\n" +
			"  Deploy LLM models to your cloud with one command.\n" +
			"  Your data never leaves your infrastructure.",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.PersistentFlags().StringVar(&providerName, "provider", "aws", "Cloud provider to use (aws)")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed provider resource events")

	root.AddCommand(newLoginCmd(&providerName))
	root.AddCommand(newDeployCmd(&providerName, &verbose))
	root.AddCommand(newDestroyCmd(&providerName, &verbose))
	root.AddCommand(newStatusCmd(&providerName))
	root.AddCommand(newCertCmd(&providerName))
	root.AddCommand(newChatCmd(&providerName))
	root.AddCommand(newStopCmd(&providerName))
	root.AddCommand(newStartCmd(&providerName))
	root.AddCommand(newCostCmd(&providerName))

	return root
}

func Execute() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		if errors.Is(err, provider.ErrNoAccount) {
			return
		}
		if errors.Is(err, provider.ErrNotLoggedIn) {
			fmt.Fprintf(os.Stderr, "\033[31m%v\033[0m\n", err)
			fmt.Fprintf(os.Stderr, "\n  Run: \033[33mhaven login\033[0m\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "\033[31merror: %v\033[0m\n", err)
		os.Exit(1)
	}
}

func buildProvider(ctx context.Context, name string, out io.Writer) (provider.Provider, provider.StateStore, error) {
	switch name {
	case "aws":
		return awsprovider.ResumeSession(ctx, out)
	default:
		return nil, nil, fmt.Errorf("unknown provider %q - available: aws", name)
	}
}
