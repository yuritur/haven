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

//go:embed llamacpp.sh
var llamacppScript string

type BootstrapInput struct {
	Runtime models.Runtime
	Tag     string
	APIKey  string
	TLSCert string
	TLSKey  string
	HFRepo  string
	HFFile  string
	GPU     bool
}

func Generate(input BootstrapInput) (string, error) {
	if input.TLSCert == "" || input.TLSKey == "" {
		return "", fmt.Errorf("TLS cert and key are required")
	}
	certB64 := base64.StdEncoding.EncodeToString([]byte(input.TLSCert))
	keyB64 := base64.StdEncoding.EncodeToString([]byte(input.TLSKey))

	switch input.Runtime {
	case models.RuntimeOllama:
		r := strings.NewReplacer(
			"{{HAVEN_MODEL}}", input.Tag,
			"{{HAVEN_API_KEY}}", input.APIKey,
			"{{HAVEN_TLS_CERT_B64}}", certB64,
			"{{HAVEN_TLS_KEY_B64}}", keyB64,
		)
		return r.Replace(ollamaScript), nil

	case models.RuntimeLlamaCpp:
		gpuLayers := ""
		if input.GPU {
			gpuLayers = "--n-gpu-layers -1"
		}
		r := strings.NewReplacer(
			"{{HAVEN_HF_REPO}}", input.HFRepo,
			"{{HAVEN_HF_FILE}}", input.HFFile,
			"{{HAVEN_API_KEY}}", input.APIKey,
			"{{HAVEN_TLS_CERT_B64}}", certB64,
			"{{HAVEN_TLS_KEY_B64}}", keyB64,
			"{{HAVEN_GPU_LAYERS}}", gpuLayers,
		)
		return r.Replace(llamacppScript), nil

	default:
		return "", fmt.Errorf("unsupported runtime %q", input.Runtime)
	}
}
