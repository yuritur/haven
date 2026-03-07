package bootstrap

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/havenapp/haven/internal/models"
)

//go:embed ollama.sh
var ollamaScript string

func Generate(runtime models.Runtime, tag, apiKey string) (string, error) {
	switch runtime {
	case models.RuntimeOllama:
		r := strings.NewReplacer(
			"{{HAVEN_MODEL}}", tag,
			"{{HAVEN_API_KEY}}", apiKey,
		)
		return r.Replace(ollamaScript), nil
	default:
		return "", fmt.Errorf("unsupported runtime %q", runtime)
	}
}
