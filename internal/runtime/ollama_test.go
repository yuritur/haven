package runtime

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/havenapp/haven/internal/models"
)

func init() {
	pollInterval = 10 * time.Millisecond
}

func TestOllamaWaitForReady_Success(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing auth header")
		}
		if n < 3 {
			w.WriteHeader(503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:1b"}]}`))
	}))
	defer srv.Close()

	rt := &OllamaRuntime{}
	err := rt.waitForReadyWithClient(context.Background(), srv.Client(), srv.URL, "llama3.2:1b", "test-key", io.Discard, 5*time.Second)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if atomic.LoadInt32(&calls) < 3 {
		t.Fatalf("expected at least 3 poll attempts, got %d", calls)
	}
}

func TestOllamaWaitForReady_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	rt := &OllamaRuntime{}
	err := rt.waitForReadyWithClient(context.Background(), srv.Client(), srv.URL, "llama3.2:1b", "test-key", io.Discard, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if got := err.Error(); got != "timed out after 50ms" {
		t.Fatalf("unexpected error: %v", got)
	}
}

func TestOllamaWaitForReady_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	rt := &OllamaRuntime{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := rt.waitForReadyWithClient(ctx, srv.Client(), srv.URL, "llama3.2:1b", "test-key", io.Discard, 5*time.Second)
	if err == nil {
		t.Fatal("expected context error")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestOllamaPort(t *testing.T) {
	rt := &OllamaRuntime{}
	if rt.Port() != 11434 {
		t.Fatalf("expected 11434, got %d", rt.Port())
	}
}

func TestNewUnsupportedRuntime(t *testing.T) {
	_, err := newRuntime(models.RuntimeName("unknown"))
	if err == nil {
		t.Fatal("expected error for unsupported runtime")
	}
}

func TestNewOllamaRuntime(t *testing.T) {
	rt, err := newRuntime(models.Ollama)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Port() != 11434 {
		t.Fatalf("expected 11434, got %d", rt.Port())
	}
}
