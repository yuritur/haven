package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newCertCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "cert <id>",
		Short: "Print the TLS certificate for a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCert(cmd.Context(), *providerName, args[0])
		},
	}
}

func runCert(ctx context.Context, providerName, id string) error {
	_, store, err := buildProviderAndStore(ctx, providerName, io.Discard)
	if err != nil {
		return err
	}

	d, err := store.Load(ctx, id)
	if err != nil {
		return fmt.Errorf("load deployment: %w", err)
	}

	if d.TLSCert == "" {
		return fmt.Errorf("deployment %s has no TLS certificate", id)
	}

	fmt.Print(d.TLSCert)
	return nil
}
