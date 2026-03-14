package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/mock"
)

func TestRunCost_Basic(t *testing.T) {
	prov := &mock.Provider{}
	store := &mock.StateStore{
		LoadFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:           "haven-a1b2c3d4",
				Model:        "llama3.2:1b",
				InstanceType: "t3.large",
				InstanceID:   "i-1234567890",
				CreatedAt:    time.Now().Add(-24 * time.Hour),
			}, nil
		},
	}

	var buf bytes.Buffer
	err := runCost(context.Background(), prov, store, "haven-a1b2c3d4", false, &buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "$") {
		t.Errorf("expected output to contain '$', got: %s", out)
	}
	if !strings.Contains(out, "haven-a1b2c3d4") {
		t.Errorf("expected output to contain deployment ID, got: %s", out)
	}
}

func TestRunCost_Projected(t *testing.T) {
	prov := &mock.Provider{}
	store := &mock.StateStore{
		LoadFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:           "haven-a1b2c3d4",
				Model:        "llama3.2:1b",
				InstanceType: "t3.large",
				InstanceID:   "i-1234567890",
				CreatedAt:    time.Now().Add(-24 * time.Hour),
			}, nil
		},
	}

	var buf bytes.Buffer
	err := runCost(context.Background(), prov, store, "haven-a1b2c3d4", true, &buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Projected to end of month") {
		t.Errorf("expected projected output, got: %s", out)
	}
}

func TestRunCost_NotFound(t *testing.T) {
	prov := &mock.Provider{}
	store := &mock.StateStore{
		LoadFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return nil, fmt.Errorf("deployment not found")
		},
	}

	var buf bytes.Buffer
	err := runCost(context.Background(), prov, store, "nonexistent", false, &buf)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunCost_UnknownInstanceType(t *testing.T) {
	prov := &mock.Provider{}
	store := &mock.StateStore{
		LoadFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:           "haven-a1b2c3d4",
				Model:        "llama3.2:1b",
				InstanceType: "x99.mega",
				InstanceID:   "i-1234567890",
				CreatedAt:    time.Now().Add(-24 * time.Hour),
			}, nil
		},
	}

	var buf bytes.Buffer
	err := runCost(context.Background(), prov, store, "haven-a1b2c3d4", false, &buf)
	if err != nil {
		t.Fatalf("expected no error (warning only), got: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Warning") {
		t.Errorf("expected warning in output, got: %s", out)
	}
}
