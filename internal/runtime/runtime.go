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

func Resolve(modelName string, override models.Runtime) (Runtime, models.Runtime, error) {
	cfg, err := models.Lookup(modelName)
	if err != nil {
		return nil, "", err
	}

	var kind models.Runtime
	if override != "" {
		if !cfg.SupportsRuntime(override) {
			return nil, "", fmt.Errorf("model %q does not support runtime %q", modelName, override)
		}
		kind = override
	} else {
		switch {
		case cfg.Ollama != nil:
			kind = models.RuntimeOllama
		case cfg.LlamaCpp != nil:
			kind = models.RuntimeLlamaCpp
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

func newRuntime(r models.Runtime) (Runtime, error) {
	switch r {
	case models.RuntimeOllama:
		return &OllamaRuntime{}, nil
	case models.RuntimeLlamaCpp:
		return &LlamaCppRuntime{}, nil
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", r)
	}
}
