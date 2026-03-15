package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/certutil"
	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/provider"
	rtm "github.com/havenapp/haven/internal/runtime"
	"github.com/havenapp/haven/internal/tui"
)

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

	var history []rtm.ChatMessage
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

		history = append(history, rtm.ChatMessage{Role: "user", Content: line})

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

		history = append(history, rtm.ChatMessage{Role: "assistant", Content: reply})
	}
}

func streamChat(ctx context.Context, client *http.Client, d *provider.Deployment, history []rtm.ChatMessage) (string, error) {
	rt, _, err := rtm.Resolve(d.Model, models.RuntimeName(d.Runtime))
	if err != nil {
		return "", fmt.Errorf("resolve runtime: %w", err)
	}

	body, err := rt.MarshalChatRequest(d.Model, history)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("https://%s:11434%s", d.PublicIP, rt.ChatPath())
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
		token, done, err := rt.ParseChatToken(scanner.Bytes())
		if err != nil {
			spin.Stop()
			return "", err
		}
		if done {
			break
		}
		if token == "" {
			continue
		}
		if first {
			spin.Stop()
			fmt.Print("\033[36m")
			first = false
		}
		fmt.Print(token)
		full.WriteString(token)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if first {
		spin.Stop()
	}
	fmt.Print("\033[0m")
	fmt.Println()

	return full.String(), nil
}
