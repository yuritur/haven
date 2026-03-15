package runtime

import (
	"testing"

	"github.com/havenapp/haven/internal/models"
)

func TestResolve_Default(t *testing.T) {
	serving, kind, err := Resolve("llama3.2:1b", "")
	if err != nil {
		t.Fatal(err)
	}
	if kind != models.Ollama {
		t.Errorf("kind = %q, want %q", kind, models.Ollama)
	}
	if serving == nil {
		t.Fatal("expected non-nil Runtime")
	}
}

func TestResolve_Override(t *testing.T) {
	serving, kind, err := Resolve("llama3.2:1b", models.LlamaCpp)
	if err != nil {
		t.Fatal(err)
	}
	if kind != models.LlamaCpp {
		t.Errorf("kind = %q, want %q", kind, models.LlamaCpp)
	}
	if serving == nil {
		t.Fatal("expected non-nil Runtime")
	}
}

func TestResolve_UnsupportedRuntime(t *testing.T) {
	_, _, err := Resolve("llama3.2:1b", "vllm")
	if err == nil {
		t.Fatal("expected error for unsupported runtime")
	}
}

func TestResolve_UnknownModel(t *testing.T) {
	_, _, err := Resolve("nonexistent", "")
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
}
