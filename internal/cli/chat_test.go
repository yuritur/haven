package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/mock"
)

func TestResolveDeployment_ByID(t *testing.T) {
	d := &provider.Deployment{ID: "haven-aaaa1111", Model: "llama3.2:1b"}
	store := &mock.Provider{
		LoadDeploymentFn: func(_ context.Context, id string) (*provider.Deployment, error) {
			if id == "haven-aaaa1111" {
				return d, nil
			}
			return nil, fmt.Errorf("not found")
		},
	}

	got, err := resolveDeployment(context.Background(), store, &mock.Prompter{}, "haven-aaaa1111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != d.ID {
		t.Fatalf("got ID %q, want %q", got.ID, d.ID)
	}
}

func TestResolveDeployment_ByID_NotFound(t *testing.T) {
	store := &mock.Provider{
		LoadDeploymentFn: func(_ context.Context, id string) (*provider.Deployment, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	_, err := resolveDeployment(context.Background(), store, &mock.Prompter{}, "haven-missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveDeployment_ZeroDeployments(t *testing.T) {
	store := &mock.Provider{
		ListFn: func(_ context.Context) ([]provider.Deployment, error) {
			return []provider.Deployment{}, nil
		},
	}

	_, err := resolveDeployment(context.Background(), store, &mock.Prompter{}, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no active deployments") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestResolveDeployment_SingleDeployment(t *testing.T) {
	store := &mock.Provider{
		ListFn: func(_ context.Context) ([]provider.Deployment, error) {
			return []provider.Deployment{
				{ID: "haven-aaaa1111", Model: "llama3.2:1b"},
			}, nil
		},
	}

	got, err := resolveDeployment(context.Background(), store, &mock.Prompter{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "haven-aaaa1111" {
		t.Fatalf("got ID %q, want %q", got.ID, "haven-aaaa1111")
	}
}

func TestResolveDeployment_MultipleDeployments(t *testing.T) {
	store := &mock.Provider{
		ListFn: func(_ context.Context) ([]provider.Deployment, error) {
			return []provider.Deployment{
				{ID: "haven-aaaa1111", Model: "llama3.2:1b"},
				{ID: "haven-bbbb2222", Model: "phi3:mini"},
			}, nil
		},
	}
	prompter := &mock.Prompter{
		SelectFn: func(prompt string, options []string) int {
			return 1
		},
	}

	got, err := resolveDeployment(context.Background(), store, prompter, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "haven-bbbb2222" {
		t.Fatalf("got ID %q, want %q", got.ID, "haven-bbbb2222")
	}
}

func TestResolveDeployment_SelectCancelled(t *testing.T) {
	store := &mock.Provider{
		ListFn: func(_ context.Context) ([]provider.Deployment, error) {
			return []provider.Deployment{
				{ID: "haven-aaaa1111", Model: "llama3.2:1b"},
				{ID: "haven-bbbb2222", Model: "phi3:mini"},
			}, nil
		},
	}
	// Default mock.Prompter.Select returns -1 (no selection)
	prompter := &mock.Prompter{}

	_, err := resolveDeployment(context.Background(), store, prompter, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no deployment selected") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
