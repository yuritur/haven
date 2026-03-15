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
			_, err := bootstrap.Generate(bootstrap.BootstrapInput{
				Runtime: models.RuntimeOllama,
				Tag:     "llama3.2:1b",
				APIKey:  "sk-test",
				TLSCert: tc.tlsCert,
				TLSKey:  tc.tlsKey,
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
		Runtime: models.RuntimeOllama,
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
