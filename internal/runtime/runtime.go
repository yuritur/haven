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
