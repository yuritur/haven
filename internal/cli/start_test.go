package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/mock"
)

func TestRunStart_Success(t *testing.T) {
	startCalled := false
	saveCalled := false
	var saved provider.Deployment

	prov := &mock.Provider{
		StartFn: func(ctx context.Context, instanceID string) error {
			startCalled = true
			if instanceID != "i-abc123" {
				t.Errorf("expected instance ID i-abc123, got %s", instanceID)
			}
			return nil
		},
	}

	stopped := time.Now().Add(-2 * time.Hour)
	store := &mock.StateStore{
		LoadFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:         "haven-test1234",
				InstanceID: "i-abc123",
				StoppedAt:  &stopped,
			}, nil
		},
		SaveFn: func(ctx context.Context, d provider.Deployment) error {
			saveCalled = true
			saved = d
			return nil
		},
	}

	err := runStart(context.Background(), prov, store, "haven-test1234")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !startCalled {
		t.Error("expected prov.Start to be called")
	}
	if !saveCalled {
		t.Error("expected store.Save to be called")
	}
	if saved.StoppedAt != nil {
		t.Error("expected StoppedAt to be nil after start")
	}
	if saved.AccumulatedStopHours < 1.9 {
		t.Errorf("expected AccumulatedStopHours >= 1.9, got %f", saved.AccumulatedStopHours)
	}
}

func TestRunStart_NotStopped(t *testing.T) {
	prov := &mock.Provider{}

	store := &mock.StateStore{
		LoadFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID: "haven-test1234",
			}, nil
		},
	}

	err := runStart(context.Background(), prov, store, "haven-test1234")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not stopped") {
		t.Errorf("error %q should contain 'not stopped'", err.Error())
	}
}
