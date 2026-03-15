package runtime

import (
	"testing"

	"github.com/havenapp/haven/internal/models"
)

func TestResolve_Default(t *testing.T) {
	res, err := Resolve("llama3.2:1b", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != models.RuntimeOllama {
		t.Errorf("Kind = %q, want %q", res.Kind, models.RuntimeOllama)
	}
	if res.Runtime == nil {
		t.Fatal("expected non-nil Runtime")
	}
	if res.ModelTag == "" {
		t.Error("expected non-empty ModelTag for Ollama")
	}
}

func TestResolve_Override(t *testing.T) {
	res, err := Resolve("llama3.2:1b", models.RuntimeLlamaCpp)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != models.RuntimeLlamaCpp {
		t.Errorf("Kind = %q, want %q", res.Kind, models.RuntimeLlamaCpp)
	}
	if res.HFRepo == "" {
		t.Error("expected non-empty HFRepo for LlamaCpp")
	}
	if res.HFFile == "" {
		t.Error("expected non-empty HFFile for LlamaCpp")
	}
}

func TestResolve_UnsupportedRuntime(t *testing.T) {
	_, err := Resolve("llama3.2:1b", "vllm")
	if err == nil {
		t.Fatal("expected error for unsupported runtime")
	}
}

func TestResolve_UnknownModel(t *testing.T) {
	_, err := Resolve("nonexistent", "")
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
}
