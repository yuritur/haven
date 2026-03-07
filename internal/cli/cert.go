package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newCertCmd(providerName *string) *cobra.Command {
	var showFingerprint bool
	cmd := &cobra.Command{
		Use:   "cert <deployment-id>",
		Short: "Print the TLS certificate for a deployment",
		Long:  "Print the TLS certificate for a deployment in PEM format.\n\nUsage with OpenAI SDK:\n  haven cert <id> > cert.pem\n  SSL_CERT_FILE=cert.pem python your_script.py",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().BoolVar(&showFingerprint, "fingerprint", false, "Print SHA-256 fingerprint instead of PEM certificate")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCert(cmd.Context(), *providerName, args[0], showFingerprint)
	}
	return cmd
}

func runCert(ctx context.Context, providerName, id string, showFingerprint bool) error {
	_, store, err := buildProviderAndStore(ctx, providerName, io.Discard)
	if err != nil {
		return err
	}

	d, err := store.Load(ctx, id)
	if err != nil {
		return fmt.Errorf("load deployment: %w", err)
	}

	if showFingerprint {
		if d.TLSFingerprint == "" {
			return fmt.Errorf("deployment %s has no TLS fingerprint", id)
		}
		fmt.Println(d.TLSFingerprint)
		return nil
	}

	if d.TLSCert == "" {
		return fmt.Errorf("deployment %s has no TLS certificate", id)
	}

	fmt.Print(d.TLSCert) // PEM already ends with a newline
	return nil
}
