package runtime

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/havenapp/haven/internal/models"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Runtime interface {
	// WaitForReady polls the runtime's health/readiness endpoint until the model is serving.
	WaitForReady(ctx context.Context, endpoint, model, apiKey, tlsFingerprint string, verbose io.Writer, timeout time.Duration) error
	// Port returns the port the runtime listens on (e.g. 11434 for both Ollama and llama.cpp behind nginx).
	Port() int
	// ChatPath returns the HTTP path for the streaming chat endpoint (e.g. "/api/chat" or "/v1/chat/completions").
	ChatPath() string
	// MarshalChatRequest serializes a chat request into the runtime's wire format.
	MarshalChatRequest(model string, history []ChatMessage) ([]byte, error)
	// ParseChatToken extracts the next content token from a single streamed line.
	// Returns empty token with done=false to skip the line, done=true on stream end.
	ParseChatToken(line []byte) (token string, done bool, err error)
}

func Resolve(modelName string, override models.RuntimeName) (Runtime, models.RuntimeName, error) {
	cfg, err := models.Lookup(modelName)
	if err != nil {
		return nil, "", err
	}

	var kind models.RuntimeName
	if override != "" {
		if !cfg.SupportsRuntime(override) {
			return nil, "", fmt.Errorf("model %q does not support runtime %q", modelName, override)
		}
		kind = override
	} else {
		switch {
		case cfg.LlamaCpp != nil:
			kind = models.LlamaCpp
		case cfg.Ollama != nil:
			kind = models.Ollama
		default:
			return nil, "", fmt.Errorf("model %q has no supported runtime", modelName)
		}
	}

	rt, err := newRuntime(kind)
	if err != nil {
		return nil, "", err
	}
	return rt, kind, nil
}

func newRuntime(r models.RuntimeName) (Runtime, error) {
	switch r {
	case models.Ollama:
		return &OllamaRuntime{}, nil
	case models.LlamaCpp:
		return &LlamaCppRuntime{}, nil
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", r)
	}
}
