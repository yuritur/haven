package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/mock"
)

func TestRunDestroy_Success(t *testing.T) {
	destroyCalled := false
	deleteCalled := false

	prov := &mock.Provider{
		DestroyFn: func(ctx context.Context, providerRef string) error {
			destroyCalled = true
			return nil
		},
	}

	store := &mock.StateStore{
		LoadFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:          "haven-test1234",
				ProviderRef: "stack-test",
				Model:       "llama3.2:1b",
				InstanceType: "t3.large",
			}, nil
		},
		DeleteFn: func(ctx context.Context, id string) error {
			deleteCalled = true
			return nil
		},
	}

	err := runDestroy(context.Background(), prov, store, "haven-test1234", true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !destroyCalled {
		t.Error("expected prov.Destroy to be called")
	}
	if !deleteCalled {
		t.Error("expected store.Delete to be called")
	}
}

func TestRunDestroy_NotFound(t *testing.T) {
	destroyCalled := false

	prov := &mock.Provider{
		DestroyFn: func(ctx context.Context, providerRef string) error {
			destroyCalled = true
			return nil
		},
	}

	store := &mock.StateStore{
		LoadFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return nil, fmt.Errorf("deployment not found")
		},
	}

	err := runDestroy(context.Background(), prov, store, "nonexistent", true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if destroyCalled {
		t.Error("expected prov.Destroy NOT to be called")
	}
}

func TestRunDestroy_DestroyFails(t *testing.T) {
	deleteCalled := false

	prov := &mock.Provider{
		DestroyFn: func(ctx context.Context, providerRef string) error {
			return fmt.Errorf("cloud API error")
		},
	}

	store := &mock.StateStore{
		LoadFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:           "haven-test1234",
				ProviderRef:  "stack-test",
				Model:        "llama3.2:1b",
				InstanceType: "t3.large",
			}, nil
		},
		DeleteFn: func(ctx context.Context, id string) error {
			deleteCalled = true
			return nil
		},
	}

	err := runDestroy(context.Background(), prov, store, "haven-test1234", true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if deleteCalled {
		t.Error("expected store.Delete NOT to be called")
	}
}
