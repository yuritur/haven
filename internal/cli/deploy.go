package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
			return runDeploy(cmd.Context(), *providerName, args[0], *verbose)
		},
	}
}

func runDeploy(ctx context.Context, providerName, modelName string, verbose bool) error {
	modelCfg, err := models.Lookup(modelName)
	if err != nil {
		return err
	}

	var out io.Writer = io.Discard
	if verbose {
		out = os.Stdout
	}

	prov, store, err := buildProviderAndStore(ctx, providerName, out)
	if err != nil {
		return err
	}

	identity, err := prov.Identity(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Provider: %s  Account: %s  Region: %s\n", providerName, identity.AccountID, identity.Region)

	userIP, err := detectPublicIP()
	if err != nil {
		return fmt.Errorf("detect public IP: %w", err)
	}
	fmt.Printf("Restricting port 11434 to: %s\n\n", userIP)

	apiKey, err := generateAPIKey()
	if err != nil {
		return fmt.Errorf("generate API key: %w", err)
	}

	deploymentID, err := generateDeploymentID()
	if err != nil {
		return fmt.Errorf("generate deployment ID: %w", err)
	}

	fmt.Printf("Deploying %s on %s (id: %s)...\n\n", modelName, modelCfg.InstanceType, deploymentID)

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
		ID:           deploymentID,
		Provider:     providerName,
		ProviderRef:  result.ProviderRef,
		CreatedAt:    time.Now().UTC(),
		Region:       identity.Region,
		Model:        modelName,
		InstanceType: modelCfg.InstanceType,
		InstanceID:   result.InstanceID,
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

	if err := waitForOllama(sigCtx, endpoint, modelName, apiKey, tlsFingerprint); err != nil {
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

	fmt.Printf("\nDeployment ready!\n")
	fmt.Printf("  Endpoint : %s\n", deployment.Endpoint)
	fmt.Printf("  API Key  : %s\n", deployment.APIKey)
	fmt.Printf("  TLS Cert : haven cert %s\n", deployment.ID)
	fmt.Printf("  ID       : %s\n\n", deployment.ID)
	fmt.Printf("Test:\n")
	fmt.Printf("  haven cert %s > cert.pem\n", deployment.ID)
	fmt.Printf("  curl --cacert cert.pem %s/chat/completions \\\n", deployment.Endpoint)
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

func waitForOllama(ctx context.Context, endpoint, model, apiKey, fingerprint string) error {
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: certutil.NewPinnedTransport(fingerprint),
	}
	deadline := time.Now().Add(15 * time.Minute)

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
		if err == nil && resp.StatusCode == 200 {
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
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
	return fmt.Errorf("timed out after 15 minutes")
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
