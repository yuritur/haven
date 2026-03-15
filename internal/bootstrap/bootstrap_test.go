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
		runtime models.RuntimeName
		tlsCert string
		tlsKey  string
	}{
		{"ollama empty cert", models.Ollama, "", "somekey"},
		{"ollama empty key", models.Ollama, "somecert", ""},
		{"ollama both empty", models.Ollama, "", ""},
		{"llamacpp empty cert", models.LlamaCpp, "", "somekey"},
		{"llamacpp empty key", models.LlamaCpp, "somecert", ""},
		{"llamacpp both empty", models.LlamaCpp, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := bootstrap.Generate(bootstrap.BootstrapInput{
				Runtime: tc.runtime,
				Tag:     "llama3.2:1b",
				APIKey:  "sk-test",
				TLSCert: tc.tlsCert,
				TLSKey:  tc.tlsKey,
				HFRepo:  "bartowski/Llama-3.2-1B-Instruct-GGUF",
				HFFile:  "Llama-3.2-1B-Instruct-Q4_K_M.gguf",
			})
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

	script, err := bootstrap.Generate(bootstrap.BootstrapInput{
		Runtime: models.Ollama,
		Tag:     tag,
		APIKey:  apiKey,
		TLSCert: tlsCert,
		TLSKey:  tlsKey,
	})
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

func TestGenerate_LlamaCpp_ContainsSubstitutions(t *testing.T) {
	hfRepo := "bartowski/Llama-3.2-1B-Instruct-GGUF"
	hfFile := "Llama-3.2-1B-Instruct-Q4_K_M.gguf"
	apiKey := "sk-haven-test"
	tlsCert := "FAKE_CERT_DATA"
	tlsKey := "FAKE_KEY_DATA"

	script, err := bootstrap.Generate(bootstrap.BootstrapInput{
		Runtime: models.LlamaCpp,
		Tag:     "llama3.2:1b",
		APIKey:  apiKey,
		TLSCert: tlsCert,
		TLSKey:  tlsKey,
		HFRepo:  hfRepo,
		HFFile:  hfFile,
		GPU:     false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	unreplaced := regexp.MustCompile(`\{\{HAVEN_[^}]+\}\}`)
	if unreplaced.MatchString(script) {
		t.Errorf("script contains unreplaced placeholders: %s", unreplaced.FindString(script))
	}

	certB64 := base64.StdEncoding.EncodeToString([]byte(tlsCert))
	keyB64 := base64.StdEncoding.EncodeToString([]byte(tlsKey))

	for _, want := range []string{certB64, keyB64, hfRepo, hfFile, apiKey} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing expected value %q", want)
		}
	}
}

func TestGenerate_LlamaCpp_EmptyHF(t *testing.T) {
	cases := []struct {
		name   string
		hfRepo string
		hfFile string
	}{
		{"empty repo", "", "model.gguf"},
		{"empty file", "org/repo", ""},
		{"both empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			script, err := bootstrap.Generate(bootstrap.BootstrapInput{
				Runtime: models.LlamaCpp,
				Tag:     "llama3.2:1b",
				APIKey:  "sk-test",
				TLSCert: "cert",
				TLSKey:  "key",
				HFRepo:  tc.hfRepo,
				HFFile:  tc.hfFile,
			})
			// Even with empty HF fields, Generate itself doesn't validate them —
			// it just substitutes. The placeholders will be empty strings in output.
			// If Generate returns no error, verify the script is non-empty.
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if script == "" {
				t.Fatal("expected non-empty script")
			}
		})
	}
}

func TestGenerate_UnsupportedRuntime(t *testing.T) {
	_, err := bootstrap.Generate(bootstrap.BootstrapInput{
		Runtime: "vllm",
		Tag:     "llama3.2:1b",
		APIKey:  "sk-test",
		TLSCert: "cert",
		TLSKey:  "key",
	})
	if err == nil {
		t.Fatal("expected error for unsupported runtime, got nil")
	}
}
