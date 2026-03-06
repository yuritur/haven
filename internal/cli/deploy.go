package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/provider"
)

func newDeployCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:     "deploy <model>",
		Short:   "Deploy a model to your cloud",
		Example: "  haven deploy llama3.2:1b\n  haven deploy phi3:mini --provider aws",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(cmd.Context(), *providerName, args[0])
		},
	}
}

func runDeploy(ctx context.Context, providerName, modelName string) error {
	modelCfg, err := models.Lookup(modelName)
	if err != nil {
		return err
	}

	prov, store, err := buildProviderAndStore(ctx, providerName)
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

	result, err := prov.Deploy(ctx, provider.DeployInput{
		DeploymentID: deploymentID,
		Model:        modelCfg.OllamaTag,
		InstanceType: modelCfg.InstanceType,
		UserIP:       userIP + "/32",
		APIKey:       apiKey,
	})
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	endpoint := fmt.Sprintf("http://%s:11434", result.PublicIP)
	fmt.Printf("\nInstance up at %s. Waiting for Ollama + model pull...\n", result.PublicIP)

	if err := waitForOllama(ctx, endpoint, modelName, apiKey); err != nil {
		return fmt.Errorf("waiting for Ollama: %w", err)
	}

	deployment := provider.Deployment{
		ID:           deploymentID,
		Provider:     providerName,
		ProviderRef:  result.ProviderRef,
		CreatedAt:    time.Now().UTC(),
		Region:       identity.Region,
		Model:        modelName,
		InstanceType: modelCfg.InstanceType,
		InstanceID:   result.InstanceID,
		PublicIP:     result.PublicIP,
		Endpoint:     endpoint + "/v1",
		APIKey:       apiKey,
	}

	if err := store.Save(ctx, deployment); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	fmt.Printf("\nDeployment ready!\n")
	fmt.Printf("  Endpoint : %s\n", deployment.Endpoint)
	fmt.Printf("  API Key  : %s\n", deployment.APIKey)
	fmt.Printf("  ID       : %s\n\n", deployment.ID)
	fmt.Printf("Test:\n")
	fmt.Printf("  curl %s/chat/completions \\\n", deployment.Endpoint)
	fmt.Printf("    -H 'Authorization: Bearer %s' \\\n", deployment.APIKey)
	fmt.Printf("    -H 'Content-Type: application/json' \\\n")
	fmt.Printf("    -d '{\"model\":\"%s\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}]}'\n", modelName)
	return nil
}

func detectPublicIP() (string, error) {
	resp, err := http.Get("https://checkip.amazonaws.com/")
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

func waitForOllama(ctx context.Context, endpoint, model, apiKey string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	deadline := time.Now().Add(15 * time.Minute)
	modelBase := strings.SplitN(model, ":", 2)[0]

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
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 && strings.Contains(string(body), modelBase) {
				fmt.Println(" ready!")
				return nil
			}
		}
		fmt.Print(".")
		time.Sleep(10 * time.Second)
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
