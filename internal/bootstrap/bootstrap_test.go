package bootstrap_test

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"

	"github.com/havenapp/haven/internal/bootstrap"
	"github.com/havenapp/haven/internal/models"
)

func TestGenerate_EmptyTLS(t *testing.T) {
	cases := []struct {
		name    string
		tlsCert string
		tlsKey  string
	}{
		{"empty cert", "", "somekey"},
		{"empty key", "somecert", ""},
		{"both empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := bootstrap.Generate(models.RuntimeOllama, "llama3.2:1b", "sk-test", tc.tlsCert, tc.tlsKey, false)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestGenerate_ContainsSubstitutions(t *testing.T) {
	tag := "llama3.2:1b"
	apiKey := "sk-haven-test"
	tlsCert := "FAKE_CERT_DATA"
	tlsKey := "FAKE_KEY_DATA"

	script, err := bootstrap.Generate(models.RuntimeOllama, tag, apiKey, tlsCert, tlsKey, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	unreplaced := regexp.MustCompile(`\{\{HAVEN_[^}]+\}\}`)
	if unreplaced.MatchString(script) {
		t.Errorf("script contains unreplaced placeholders: %s", unreplaced.FindString(script))
	}

	certB64 := base64.StdEncoding.EncodeToString([]byte(tlsCert))
	keyB64 := base64.StdEncoding.EncodeToString([]byte(tlsKey))

	for _, want := range []string{certB64, keyB64, tag, apiKey} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing expected value %q", want)
		}
	}
}

func TestGenerate_UnsupportedRuntime(t *testing.T) {
	// TLS values are non-empty so the TLS guard does not fire before the runtime check.
	_, err := bootstrap.Generate("vllm", "llama3.2:1b", "sk-test", "cert", "key", false)
	if err == nil {
		t.Fatal("expected error for unsupported runtime, got nil")
	}
}

func TestGenerate_GPUScript(t *testing.T) {
	script, err := bootstrap.Generate(models.RuntimeOllama, "qwen3.5:4b", "sk-test", "CERT", "KEY", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"nvidia-smi", "cuda-toolkit"} {
		if !strings.Contains(script, want) {
			t.Errorf("GPU script missing expected string %q", want)
		}
	}
}

func TestGenerate_CPUScript(t *testing.T) {
	script, err := bootstrap.Generate(models.RuntimeOllama, "llama3.2:1b", "sk-test", "CERT", "KEY", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(script, `"false" = "true"`) {
		t.Error("CPU script should have GPU conditional evaluate to false")
	}
	if strings.Contains(script, `"true" = "true"`) {
		t.Error("CPU script should not have GPU conditional evaluate to true")
	}
}
