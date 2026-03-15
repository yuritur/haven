package runtime

import (
	"testing"

	"github.com/havenapp/haven/internal/models"
)

func TestResolve_Default(t *testing.T) {
	_, rt, err := Resolve("llama3.2:1b", "")
	if err != nil {
		t.Fatal(err)
	}
	if rt != models.RuntimeOllama {
		t.Errorf("default runtime = %q, want %q", rt, models.RuntimeOllama)
	}
}

func TestResolve_Override(t *testing.T) {
	cfg, rt, err := Resolve("llama3.2:1b", models.RuntimeLlamaCpp)
	if err != nil {
		t.Fatal(err)
	}
	if rt != models.RuntimeLlamaCpp {
		t.Errorf("runtime = %q, want %q", rt, models.RuntimeLlamaCpp)
	}
	if cfg.LlamaCpp == nil {
		t.Fatal("expected non-nil LlamaCpp config")
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
