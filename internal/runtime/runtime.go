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

type Resolved struct {
	Runtime  Runtime
	Kind     models.Runtime
	ModelTag string
	HFRepo   string
	HFFile   string
}

func Resolve(modelName string, override models.Runtime) (Resolved, error) {
	cfg, err := models.Lookup(modelName)
	if err != nil {
		return Resolved{}, err
	}

	var kind models.Runtime
	if override != "" {
		if !cfg.SupportsRuntime(override) {
			return Resolved{}, fmt.Errorf("model %q does not support runtime %q", modelName, override)
		}
		kind = override
	} else {
		switch {
		case cfg.Ollama != nil:
			kind = models.RuntimeOllama
		case cfg.LlamaCpp != nil:
			kind = models.RuntimeLlamaCpp
		default:
			return Resolved{}, fmt.Errorf("model %q has no supported runtime", modelName)
		}
	}

	rt, err := newRuntime(kind)
	if err != nil {
		return Resolved{}, err
	}

	res := Resolved{Runtime: rt, Kind: kind}
	switch kind {
	case models.RuntimeOllama:
		res.ModelTag = cfg.Ollama.Tag
	case models.RuntimeLlamaCpp:
		res.HFRepo = cfg.LlamaCpp.HFRepo
		res.HFFile = cfg.LlamaCpp.HFFile
	}
	return res, nil
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
