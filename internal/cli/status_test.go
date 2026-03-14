package cli

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/mock"
)

func TestRunStatus_Empty(t *testing.T) {
	store := &mock.StateStore{
		ListFn: func(ctx context.Context) ([]provider.Deployment, error) {
			return []provider.Deployment{}, nil
		},
	}

	err := runStatus(context.Background(), store)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestRunStatus_Multiple(t *testing.T) {
	store := &mock.StateStore{
		ListFn: func(ctx context.Context) ([]provider.Deployment, error) {
			return []provider.Deployment{
				{ID: "haven-aaaa1111", Provider: "aws", Model: "llama3.2:1b", InstanceType: "t3.large", Endpoint: "https://1.2.3.4:11434/v1", CreatedAt: time.Now().Add(-24 * time.Hour)},
				{ID: "haven-bbbb2222", Provider: "aws", Model: "phi3:mini", InstanceType: "t3.large", Endpoint: "https://5.6.7.8:11434/v1", CreatedAt: time.Now().Add(-48 * time.Hour)},
			}, nil
		},
	}

	err := runStatus(context.Background(), store)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestRunStatus_ListError(t *testing.T) {
	store := &mock.StateStore{
		ListFn: func(ctx context.Context) ([]provider.Deployment, error) {
			return nil, fmt.Errorf("S3 access denied")
		},
	}

	err := runStatus(context.Background(), store)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
