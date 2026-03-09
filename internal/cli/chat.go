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

func newChatCmd(providerName *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "chat [deployment-id]",
		Short:   "Interactive chat with a deployed model",
		Example: "  haven chat\n  haven chat haven-a1b2c3d4",
		Args:    cobra.MaximumNArgs(1),
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		prompter := newTerminalPrompter()
		_, store, err := buildProvider(cmd.Context(), *providerName, prompter, io.Discard)
		if err != nil {
			return err
		}
		var id string
		if len(args) == 1 {
			id = args[0]
		}
		return runChat(cmd.Context(), store, prompter, id)
	}
	return cmd
}

func runChat(ctx context.Context, store provider.StateStore, prompter provider.Prompter, id string) error {
	d, err := resolveDeployment(ctx, store, prompter, id)
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

func resolveDeployment(ctx context.Context, store provider.StateStore, prompter provider.Prompter, id string) (*provider.Deployment, error) {
	if id != "" {
		d, err := store.Load(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("load deployment: %w", err)
		}
		return d, nil
	}

	deployments, err := store.List(ctx)
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

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk chatStreamResponse
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			return "", fmt.Errorf("decode stream: %w", err)
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
	fmt.Println()

	return full.String(), nil
}
