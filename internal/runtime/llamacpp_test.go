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

func TestLlamaCppWaitForReady_Success(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	rt := &LlamaCppRuntime{}
	err := rt.waitForReadyWithClient(context.Background(), srv.Client(), srv.URL, "test-key", io.Discard, 5*time.Second)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if atomic.LoadInt32(&calls) < 3 {
		t.Fatalf("expected at least 3 poll attempts, got %d", calls)
	}
}

func TestLlamaCppWaitForReady_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	rt := &LlamaCppRuntime{}
	err := rt.waitForReadyWithClient(context.Background(), srv.Client(), srv.URL, "test-key", io.Discard, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if got := err.Error(); got != "timed out after 50ms" {
		t.Fatalf("unexpected error: %v", got)
	}
}

func TestLlamaCppWaitForReady_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	rt := &LlamaCppRuntime{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := rt.waitForReadyWithClient(ctx, srv.Client(), srv.URL, "test-key", io.Discard, 5*time.Second)
	if err == nil {
		t.Fatal("expected context error")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestLlamaCppPort(t *testing.T) {
	rt := &LlamaCppRuntime{}
	if rt.Port() != 11434 {
		t.Fatalf("expected 11434, got %d", rt.Port())
	}
}

func TestNewLlamaCppRuntime(t *testing.T) {
	rt, err := New(models.RuntimeLlamaCpp)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Port() != 11434 {
		t.Fatalf("expected 11434, got %d", rt.Port())
	}
}
