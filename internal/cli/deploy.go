package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/certutil"
	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/runtime"
	"github.com/havenapp/haven/internal/tui"
)

func newDeployCmd(providerName *string, verbose *bool) *cobra.Command {
	var runtimeFlag string
	cmd := &cobra.Command{
		Use:     "deploy <model>",
		Short:   "Deploy a model to your cloud",
		Example: "  haven deploy llama3.2:1b\n  haven deploy phi3:mini --runtime llamacpp",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing model name\n\n  Usage: haven deploy <model>\n  Example: haven deploy llama3.2:1b")
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var out io.Writer = io.Discard
			if *verbose {
				out = os.Stdout
			}
			prompter := newTerminalPrompter()
			prov, err := buildProvider(cmd.Context(), *providerName, out)
			if err != nil {
				return err
			}
			return runDeploy(cmd.Context(), prov, *providerName, args[0], runtimeFlag, *verbose, out, prompter)
		},
	}
	cmd.Flags().StringVar(&runtimeFlag, "runtime", "", "serving runtime: ollama (default) or llamacpp")
	return cmd
}

func runDeploy(ctx context.Context, prov provider.Provider, providerName string, modelName string, runtimeFlag string, verbose bool, out io.Writer, prompter provider.Prompter) error {
	serving, runtimeKind, err := runtime.Resolve(modelName, models.Runtime(runtimeFlag))
	if err != nil {
		return err
	}

	identity, err := prov.Identity(ctx)
	if err != nil {
		return err
	}

	userIP, err := detectPublicIP()
	if err != nil {
		return fmt.Errorf("detect public IP: %w", err)
	}
	fmt.Printf("\033[33mRestricting port 11434 to:\033[0m %s\n\n", userIP)

	existing, err := prov.List(ctx)
	if err != nil {
		return fmt.Errorf("check existing deployments: %w", err)
	}
	if len(existing) > 0 {
		return fmt.Errorf("deployment %s is already active — run `haven destroy %s` first\n\nNote: multiple simultaneous deployments are not yet supported but will be in a future release", existing[0].ID, existing[0].ID)
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return fmt.Errorf("generate API key: %w", err)
	}

	deploymentID, err := generateDeploymentID()
	if err != nil {
		return fmt.Errorf("generate deployment ID: %w", err)
	}

	err = prov.EnsureQuota(ctx, modelName, runtimeKind, prompter)
	switch {
	case errors.Is(err, provider.ErrQuotaUserExit):
		return nil
	case err != nil:
		return err
	}

	fmt.Printf("\033[33mDeploying\033[0m %s (id: %s)...\n\n", modelName, deploymentID)

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cleanup := func(providerRef string) {
		stop() // let subsequent Ctrl+C kill the process immediately
		fmt.Printf("\nInterrupted — destroying %s...\n", providerRef)
		if err := prov.Destroy(context.Background(), providerRef); err != nil {
			fmt.Fprintf(os.Stderr, "destroy: %v\n", err)
		}
	}

	var spin *tui.Spinner
	if !verbose {
		spin = tui.StartSpinner("Provisioning infrastructure...")
	}

	tlsCert, tlsKey, tlsFingerprint, err := certutil.GenerateSelfSigned()
	if err != nil {
		return fmt.Errorf("generate TLS cert: %w", err)
	}

	result, err := prov.Deploy(sigCtx, provider.DeployInput{
		DeploymentID:   deploymentID,
		Runtime:        runtimeKind,
		Model:          modelName,
		UserIP:         userIP + "/32",
		APIKey:         apiKey,
		TLSCert:        tlsCert,
		TLSKey:         tlsKey,
		TLSFingerprint: tlsFingerprint,
	})

	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		if sigCtx.Err() != nil {
			cleanup(deploymentID)
		}
		return fmt.Errorf("deploy: %w", err)
	}

	fmt.Printf("\033[33mUsing instance:\033[0m %s\n", result.InstanceType)

	endpoint := fmt.Sprintf("https://%s:%d", result.PublicIP, serving.Port())

	deployment := provider.Deployment{
		ID:             deploymentID,
		Provider:       providerName,
		Runtime:        string(runtimeKind),
		ProviderRef:    result.ProviderRef,
		CreatedAt:      time.Now().UTC(),
		Region:         identity.Region,
		Model:          modelName,
		InstanceType:   result.InstanceType,
		InstanceID:     result.InstanceID,
		PublicIP:       result.PublicIP,
		Endpoint:       endpoint + "/v1",
		APIKey:         apiKey,
		TLSCert:        tlsCert,
		TLSFingerprint: tlsFingerprint,
	}

	if err := prov.SaveDeployment(ctx, deployment); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	if runtimeKind == models.RuntimeLlamaCpp {
		fmt.Printf("Instance up at %s. Waiting for model...\n", result.PublicIP)
	} else {
		fmt.Printf("Instance up at %s. Pulling model...\n", result.PublicIP)
	}

	if !verbose {
		spin = tui.StartSpinner("Waiting for model to be ready...")
	}

	pollTimeout := 15 * time.Minute
	if result.GPU {
		pollTimeout = 30 * time.Minute
	}

	if err := serving.WaitForReady(sigCtx, endpoint, modelName, apiKey, tlsFingerprint, out, pollTimeout); err != nil {
		if spin != nil {
			spin.Stop()
		}
		if sigCtx.Err() != nil {
			cleanup(result.ProviderRef)
			_ = prov.DeleteDeployment(context.Background(), deploymentID)
			return nil
		}
		return fmt.Errorf("waiting for model: %w\nDeployment %s is saved — run `haven destroy %s` to clean up", err, deploymentID, deploymentID)
	}

	if spin != nil {
		spin.Stop()
	}

	certDir := "data/certs"
	if err := os.MkdirAll(certDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create %s: %v\n", certDir, err)
	}
	certFile := certDir + "/" + deploymentID + ".pem"
	if err := os.WriteFile(certFile, []byte(tlsCert), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save %s: %v\n", certFile, err)
	}

	fmt.Printf("\n\033[33mDeployment ready!\033[0m\n")
	fmt.Printf("  Endpoint : %s\n", deployment.Endpoint)
	fmt.Printf("  API Key  : %s\n", deployment.APIKey)
	fmt.Printf("  TLS Cert : %s\n", certFile)
	fmt.Printf("  ID       : %s\n\n", deployment.ID)
	fmt.Printf("Use \033[33m`haven chat`\033[0m to talk to the model, or call the API directly:\n")
	fmt.Printf("  curl -k --cacert %s %s/chat/completions \\\n", certFile, deployment.Endpoint)
	fmt.Printf("    -H 'Authorization: Bearer %s' \\\n", deployment.APIKey)
	fmt.Printf("    -H 'Content-Type: application/json' \\\n")
	fmt.Printf("    -d '{\"model\":\"%s\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}]}'\n", modelName)
	return nil
}

func detectPublicIP() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://checkip.amazonaws.com/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func generateAPIKey() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk-haven-" + hex.EncodeToString(b), nil
}

func generateDeploymentID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "haven-" + hex.EncodeToString(b), nil
}
