package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/mock"
)

func TestRunStop_Success(t *testing.T) {
	stopCalled := false
	saveCalled := false
	var saved provider.Deployment

	prov := &mock.Provider{
		StopFn: func(ctx context.Context, instanceID string) error {
			stopCalled = true
			if instanceID != "i-abc123" {
				t.Errorf("expected instance ID i-abc123, got %s", instanceID)
			}
			return nil
		},
	}

	store := &mock.StateStore{
		LoadFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:         "haven-test1234",
				InstanceID: "i-abc123",
			}, nil
		},
		SaveFn: func(ctx context.Context, d provider.Deployment) error {
			saveCalled = true
			saved = d
			return nil
		},
	}

	err := runStop(context.Background(), prov, store, "haven-test1234")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !stopCalled {
		t.Error("expected prov.Stop to be called")
	}
	if !saveCalled {
		t.Error("expected store.Save to be called")
	}
	if saved.StoppedAt == nil {
		t.Error("expected StoppedAt to be set")
	}
}

func TestRunStop_AlreadyStopped(t *testing.T) {
	prov := &mock.Provider{}
	stopped := time.Now()

	store := &mock.StateStore{
		LoadFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:        "haven-test1234",
				StoppedAt: &stopped,
			}, nil
		},
	}

	err := runStop(context.Background(), prov, store, "haven-test1234")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "already stopped") {
		t.Errorf("error %q should contain 'already stopped'", err.Error())
	}
}
