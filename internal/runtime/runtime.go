package runtime

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/havenapp/haven/internal/models"
)

type Runtime interface {
	WaitForReady(ctx context.Context, endpoint, model, apiKey, tlsFingerprint string, verbose io.Writer, timeout time.Duration) error
	Port() int
}

func Resolve(modelName string, override models.Runtime) (models.Config, models.Runtime, error) {
	cfg, err := models.Lookup(modelName)
	if err != nil {
		return models.Config{}, "", err
	}
	if override != "" {
		if !cfg.SupportsRuntime(override) {
			return models.Config{}, "", fmt.Errorf("model %q does not support runtime %q", modelName, override)
		}
		return cfg, override, nil
	}
	switch {
	case cfg.Ollama != nil:
		return cfg, models.RuntimeOllama, nil
	case cfg.LlamaCpp != nil:
		return cfg, models.RuntimeLlamaCpp, nil
	default:
		return models.Config{}, "", fmt.Errorf("model %q has no supported runtime", modelName)
	}
}

func New(r models.Runtime) (Runtime, error) {
	switch r {
	case models.RuntimeOllama:
		return &OllamaRuntime{}, nil
	case models.RuntimeLlamaCpp:
		return &LlamaCppRuntime{}, nil
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", r)
	}
}
