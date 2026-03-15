package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/certutil"
	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/tui"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatStreamResponse struct {
	Message chatMessage `json:"message"`
	Done    bool        `json:"done"`
}

type openAIChatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func newChatCmd(providerName *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "chat [deployment-id]",
		Short:   "Interactive chat with a deployed model",
		Long:    "Interactive chat with a deployed model.\nIf only one deployment exists, it is selected automatically.",
		Example: "  haven chat\n  haven chat haven-a1b2c3d4",
		Args:    cobra.MaximumNArgs(1),
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		prompter := newTerminalPrompter()
		prov, err := buildProvider(cmd.Context(), *providerName, io.Discard)
		if err != nil {
			return err
		}
		var id string
		if len(args) == 1 {
			id = args[0]
		}
		return runChat(cmd.Context(), prov, prompter, id)
	}
	return cmd
}

func runChat(ctx context.Context, prov provider.Provider, prompter provider.Prompter, id string) error {
	d, err := resolveDeployment(ctx, prov, prompter, id)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout:   0, // streaming responses can take arbitrarily long
		Transport: certutil.NewPinnedTransport(d.TLSFingerprint),
	}

	fmt.Printf("\033[1m%s\033[0m @ %s\n", d.Model, d.Endpoint)
	fmt.Println("Type a message, or \"exit\" to quit.")
	fmt.Println()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	var history []chatMessage
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\033[33m> \033[0m")
		if !scanner.Scan() {
			fmt.Println()
			return nil
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "exit") || strings.EqualFold(line, "quit") {
			return nil
		}

		history = append(history, chatMessage{Role: "user", Content: line})

		reply, err := streamChat(ctx, client, d, history)
		if err != nil {
			if ctx.Err() != nil {
				fmt.Println()
				return nil
			}
			fmt.Fprintf(os.Stderr, "\033[31merror: %v\033[0m\n", err)
			// Remove the failed user message
			history = history[:len(history)-1]
			continue
		}

		history = append(history, chatMessage{Role: "assistant", Content: reply})
	}
}

func resolveDeployment(ctx context.Context, prov provider.Provider, prompter provider.Prompter, id string) (*provider.Deployment, error) {
	if id != "" {
		d, err := prov.LoadDeployment(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("load deployment: %w", err)
		}
		return d, nil
	}

	deployments, err := prov.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	if len(deployments) == 0 {
		return nil, fmt.Errorf("no active deployments — run 'haven deploy <model>' first")
	}
	if len(deployments) == 1 {
		return &deployments[0], nil
	}

	options := make([]string, len(deployments))
	for i, d := range deployments {
		options[i] = fmt.Sprintf("%s (%s)", d.ID, d.Model)
	}
	idx := prompter.Select("Select a deployment:", options)
	if idx < 0 {
		return nil, fmt.Errorf("no deployment selected")
	}
	return &deployments[idx], nil
}

func streamChat(ctx context.Context, client *http.Client, d *provider.Deployment, history []chatMessage) (string, error) {
	if d.Runtime == "llamacpp" {
		return streamChatOpenAI(ctx, client, d, history)
	}
	return streamChatOllama(ctx, client, d, history)
}

func streamChatOllama(ctx context.Context, client *http.Client, d *provider.Deployment, history []chatMessage) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:    d.Model,
		Messages: history,
	})
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("https://%s:11434/api/chat", d.PublicIP)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.APIKey)

	spin := tui.StartSpinner("\033[33mThinking...\033[0m")

	resp, err := client.Do(req)
	if err != nil {
		spin.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		spin.Stop()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var full strings.Builder
	first := true
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk chatStreamResponse
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			spin.Stop()
			return "", fmt.Errorf("decode stream: %w", err)
		}
		if first {
			spin.Stop()
			fmt.Print("\033[36m")
			first = false
		}
		if chunk.Done {
			break
		}
		fmt.Print(chunk.Message.Content)
		full.WriteString(chunk.Message.Content)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	fmt.Print("\033[0m")
	fmt.Println()

	return full.String(), nil
}

func streamChatOpenAI(ctx context.Context, client *http.Client, d *provider.Deployment, history []chatMessage) (string, error) {
	body, err := json.Marshal(openAIChatRequest{
		Model:    d.Model,
		Messages: history,
		Stream:   true,
	})
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("https://%s:11434/v1/chat/completions", d.PublicIP)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.APIKey)

	spin := tui.StartSpinner("\033[33mThinking...\033[0m")

	resp, err := client.Do(req)
	if err != nil {
		spin.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		spin.Stop()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var full strings.Builder
	first := true
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			break
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			spin.Stop()
			return "", fmt.Errorf("decode stream: %w", err)
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		content := chunk.Choices[0].Delta.Content
		if content == "" {
			continue
		}

		if first {
			spin.Stop()
			fmt.Print("\033[36m")
			first = false
		}

		fmt.Print(content)
		full.WriteString(content)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	// Stop spinner if no content was received
	if first {
		spin.Stop()
	}
	fmt.Print("\033[0m")
	fmt.Println()

	return full.String(), nil
}
