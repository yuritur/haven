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

type LlamaCppRuntime struct{}

func (l *LlamaCppRuntime) Port() int { return 11434 }

func (l *LlamaCppRuntime) WaitForReady(ctx context.Context, endpoint, model, apiKey, tlsFingerprint string, verbose io.Writer, timeout time.Duration) error {
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: certutil.NewPinnedTransport(tlsFingerprint),
	}
	return l.waitForReadyWithClient(ctx, client, endpoint, apiKey, verbose, timeout)
}

func (l *LlamaCppRuntime) waitForReadyWithClient(ctx context.Context, client *http.Client, endpoint, apiKey string, verbose io.Writer, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/health", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(verbose, "poll: %v\n", err)
		} else {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			if readErr != nil {
				fmt.Fprintf(verbose, "poll: read body: %v\n", readErr)
			} else if resp.StatusCode == 200 {
				var health struct {
					Status string `json:"status"`
				}
				if json.Unmarshal(body, &health) == nil && health.Status == "ok" {
					return nil
				}
				fmt.Fprintf(verbose, "poll: health status %q\n", health.Status)
			} else {
				fmt.Fprintf(verbose, "poll: status %d\n", resp.StatusCode)
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
