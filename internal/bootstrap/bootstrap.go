package bootstrap

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/havenapp/haven/internal/models"
)

//go:embed ollama.sh
var ollamaScript string

func Generate(runtime models.Runtime, tag, apiKey, tlsCert, tlsKey string) (string, error) {
	switch runtime {
	case models.RuntimeOllama:
		if tlsCert == "" || tlsKey == "" {
			return "", fmt.Errorf("TLS cert and key are required")
		}
		certB64 := base64.StdEncoding.EncodeToString([]byte(tlsCert))
		keyB64 := base64.StdEncoding.EncodeToString([]byte(tlsKey))
		r := strings.NewReplacer(
			"{{HAVEN_MODEL}}", tag,
			"{{HAVEN_API_KEY}}", apiKey,
			"{{HAVEN_TLS_CERT_B64}}", certB64,
			"{{HAVEN_TLS_KEY_B64}}", keyB64,
		)
		return r.Replace(ollamaScript), nil
	default:
		return "", fmt.Errorf("unsupported runtime %q", runtime)
	}
}
