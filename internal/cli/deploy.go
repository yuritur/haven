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

	havnaws "github.com/havenapp/haven/internal/aws"
	"github.com/havenapp/haven/internal/cfn"
	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/state"
)

var deployCmd = &cobra.Command{
	Use:   "deploy <model>",
	Short: "Deploy a model to AWS",
	Example: "  haven deploy llama3.2:1b\n  haven deploy phi3:mini",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeploy,
}

func runDeploy(cmd *cobra.Command, args []string) error {
	modelName := args[0]
	ctx := context.Background()

	modelCfg, err := models.Lookup(modelName)
	if err != nil {
		return err
	}

	cfg, err := havnaws.LoadConfig(ctx)
	if err != nil {
		return err
	}

	identity, err := havnaws.GetIdentity(ctx, cfg)
	if err != nil {
		return err
	}
	fmt.Printf("Account: %s  Region: %s\n", identity.AccountID, identity.Region)

	stateManager, err := state.NewManager(ctx, cfg)
	if err != nil {
		return fmt.Errorf("bootstrap state bucket: %w", err)
	}

	userIP, err := detectPublicIP()
	if err != nil {
		return fmt.Errorf("detect public IP: %w", err)
	}
	fmt.Printf("Restricting port 11434 to: %s\n\n", userIP)

	apiKey := generateAPIKey()
	stackName := generateStackName()

	fmt.Printf("Deploying %s on %s (stack: %s)...\n\n", modelName, modelCfg.InstanceType, stackName)

	result, err := cfn.Deploy(ctx, cfg, cfn.DeployInput{
		StackName: stackName,
		Model:     modelCfg.OllamaTag,
		UserIP:    userIP + "/32",
		APIKey:    apiKey,
	})
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	endpoint := fmt.Sprintf("http://%s:11434", result.PublicIP)
	fmt.Printf("\nInstance up at %s. Waiting for Ollama + model pull...\n", result.PublicIP)

	if err := waitForOllama(ctx, endpoint, modelName, apiKey); err != nil {
		return fmt.Errorf("waiting for Ollama: %w", err)
	}

	deployment := state.Deployment{
		ID:           stackName,
		CreatedAt:    time.Now().UTC(),
		Region:       identity.Region,
		StackName:    stackName,
		Model:        modelName,
		InstanceType: modelCfg.InstanceType,
		InstanceID:   result.InstanceID,
		EIP:          result.PublicIP,
		Endpoint:     endpoint + "/v1",
		APIKey:       apiKey,
	}

	if err := stateManager.Save(ctx, deployment); err != nil {
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

func generateAPIKey() string {
	b := make([]byte, 18)
	rand.Read(b)
	return "sk-haven-" + hex.EncodeToString(b)
}

func generateStackName() string {
	b := make([]byte, 4)
	rand.Read(b)
	return "haven-" + hex.EncodeToString(b)
}
