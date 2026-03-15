package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/havenapp/haven/internal/certutil"
)

type ollamaChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ollamaChatStreamResponse struct {
	Message ChatMessage `json:"message"`
	Done    bool        `json:"done"`
}

type OllamaRuntime struct{}

func (o *OllamaRuntime) Port() int { return 11434 }

func (o *OllamaRuntime) ChatPath() string { return "/api/chat" }

func (o *OllamaRuntime) MarshalChatRequest(model string, history []ChatMessage) ([]byte, error) {
	return json.Marshal(ollamaChatRequest{Model: model, Messages: history})
}

func (o *OllamaRuntime) ParseChatToken(line []byte) (string, bool, error) {
	var chunk ollamaChatStreamResponse
	if err := json.Unmarshal(line, &chunk); err != nil {
		return "", false, fmt.Errorf("decode stream: %w", err)
	}
	if chunk.Done {
		return "", true, nil
	}
	return chunk.Message.Content, false, nil
}

func (o *OllamaRuntime) WaitForReady(ctx context.Context, endpoint, model, apiKey, tlsFingerprint string, verbose io.Writer, timeout time.Duration) error {
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: certutil.NewPinnedTransport(tlsFingerprint),
	}
	return o.waitForReadyWithClient(ctx, client, endpoint, model, apiKey, verbose, timeout)
}

var pollInterval = 10 * time.Second

func (o *OllamaRuntime) waitForReadyWithClient(ctx context.Context, client *http.Client, endpoint, model, apiKey string, verbose io.Writer, timeout time.Duration) error {
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
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			if readErr != nil {
				fmt.Fprintf(verbose, "poll: read body: %v\n", readErr)
			} else {
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
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
	return fmt.Errorf("timed out after %v", timeout)
}
