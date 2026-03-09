package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	"github.com/havenapp/haven/internal/tui"
)

func newDeployCmd(providerName *string, verbose *bool) *cobra.Command {
	return &cobra.Command{
		Use:     "deploy <model>",
		Short:   "Deploy a model to your cloud",
		Example: "  haven deploy llama3.2:1b\n  haven deploy phi3:mini --provider aws",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out io.Writer = io.Discard
			if *verbose {
				out = os.Stdout
			}
			prompter := newTerminalPrompter()
			prov, store, err := authenticateProvider(cmd.Context(), *providerName, prompter, out)
			if err != nil {
				return err
			}
			promptFn := func(msg string) string {
				fmt.Print(msg)
				return prompter.Input("")
			}
			return runDeploy(cmd.Context(), prov, store, *providerName, args[0], *verbose, out, promptFn)
		},
	}
}

func runDeploy(ctx context.Context, prov provider.Provider, store provider.StateStore, providerName string, modelName string, verbose bool, out io.Writer, promptFn func(string) string) error {
	modelCfg, err := models.Lookup(modelName)
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

	existing, err := store.List(ctx)
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

	if ensurer, ok := prov.(interface {
		EnsureQuota(ctx context.Context, instanceType string, promptFn func(string) string) error
	}); ok {
		if err := ensurer.EnsureQuota(ctx, modelCfg.InstanceType, promptFn); err != nil {
			if errors.Is(err, provider.ErrQuotaUserExit) {
				return nil
			}
			return err
		}
	}

	fmt.Printf("\033[33mDeploying\033[0m %s on %s (id: %s)...\n\n", modelName, modelCfg.InstanceType, deploymentID)

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
		Runtime:        modelCfg.Runtime,
		ModelTag:       modelCfg.Tag,
		InstanceType:   modelCfg.InstanceType,
		UserIP:         userIP + "/32",
		APIKey:         apiKey,
		TLSCert:        tlsCert,
		TLSKey:         tlsKey,
		TLSFingerprint: tlsFingerprint,
		EBSVolumeGB:    modelCfg.EBSVolumeGB,
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

	endpoint := fmt.Sprintf("https://%s:11434", result.PublicIP)

	deployment := provider.Deployment{
		ID:             deploymentID,
		Provider:       providerName,
		ProviderRef:    result.ProviderRef,
		CreatedAt:      time.Now().UTC(),
		Region:         identity.Region,
		Model:          modelName,
		InstanceType:   modelCfg.InstanceType,
		InstanceID:     result.InstanceID,
		PublicIP:       result.PublicIP,
		Endpoint:       endpoint + "/v1",
		APIKey:         apiKey,
		TLSCert:        tlsCert,
		TLSFingerprint: tlsFingerprint,
	}

	if err := store.Save(ctx, deployment); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	fmt.Printf("Instance up at %s. Pulling model...\n", result.PublicIP)

	if !verbose {
		spin = tui.StartSpinner("Waiting for model to be ready...")
	}

	pollTimeout := 15 * time.Minute
	if models.IsGPUInstance(modelCfg.InstanceType) {
		pollTimeout = 30 * time.Minute
	}

	if err := waitForOllama(sigCtx, endpoint, modelName, apiKey, tlsFingerprint, out, pollTimeout); err != nil {
		if spin != nil {
			spin.Stop()
		}
		if sigCtx.Err() != nil {
			cleanup(result.ProviderRef)
			_ = store.Delete(context.Background(), deploymentID)
			return nil
		}
		return fmt.Errorf("waiting for Ollama: %w\nDeployment %s is saved — run `haven destroy %s` to clean up", err, deploymentID, deploymentID)
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

	fmt.Printf("\nDeployment ready!\n")
	fmt.Printf("  Endpoint : %s\n", deployment.Endpoint)
	fmt.Printf("  API Key  : %s\n", deployment.APIKey)
	fmt.Printf("  TLS Cert : %s\n", certFile)
	fmt.Printf("  ID       : %s\n\n", deployment.ID)
	fmt.Printf("Test:\n")
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

func waitForOllama(ctx context.Context, endpoint, model, apiKey, fingerprint string, verbose io.Writer, timeout time.Duration) error {
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: certutil.NewPinnedTransport(fingerprint),
	}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/api/tags", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(verbose, "poll: %v\n", err)
		} else if resp.StatusCode != 200 {
			resp.Body.Close()
			fmt.Fprintf(verbose, "poll: status %d\n", resp.StatusCode)
		} else {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var tagsResp struct {
				Models []struct {
					Name string `json:"name"`
				} `json:"models"`
			}
			if json.Unmarshal(body, &tagsResp) == nil {
				for _, m := range tagsResp.Models {
					if m.Name == model {
						return nil
					}
				}
			}
			fmt.Fprintf(verbose, "poll: model not yet in /api/tags\n")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
	return fmt.Errorf("timed out after %v", timeout)
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
