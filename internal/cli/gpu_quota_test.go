package cli

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/havenapp/haven/internal/provider/aws/quota"
	"github.com/havenapp/haven/internal/provider/mock"
)

func noLoadRequest(_ context.Context, _ string) (*quota.QuotaRequest, error) {
	return nil, nil
}

func TestHandleGPUQuota_Sufficient(t *testing.T) {
	checker := &mock.QuotaChecker{
		LoadGPUQuotaRequestFn: noLoadRequest,
		CheckGPUQuotaFn: func(_ context.Context, _ string) (*quota.QuotaStatus, error) {
			return &quota.QuotaStatus{
				CurrentVCPUs:  8,
				RequiredVCPUs: 4,
				Sufficient:    true,
				QuotaCode:     "L-DB2BBE81",
			}, nil
		},
	}

	result, err := handleGPUQuota(context.Background(), checker, "g5.xlarge", "us-east-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Proceed {
		t.Error("expected Proceed=true for sufficient quota")
	}
}

func TestHandleGPUQuota_InsufficientOption1(t *testing.T) {
	checker := &mock.QuotaChecker{
		LoadGPUQuotaRequestFn: noLoadRequest,
		CheckGPUQuotaFn: func(_ context.Context, _ string) (*quota.QuotaStatus, error) {
			return &quota.QuotaStatus{
				CurrentVCPUs:  0,
				RequiredVCPUs: 4,
				Sufficient:    false,
				QuotaCode:     "L-DB2BBE81",
			}, nil
		},
	}
	promptFn := func(_ string) string { return "1" }

	result, err := handleGPUQuota(context.Background(), checker, "g5.xlarge", "us-east-1", promptFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Proceed {
		t.Error("expected Proceed=false when user picks manual instructions")
	}
}

func TestHandleGPUQuota_InsufficientOption2(t *testing.T) {
	var requestCalled atomic.Bool

	// Use a context that cancels shortly after, so waitForQuotaApproval
	// returns without actually waiting 30 seconds.
	ctx, cancel := context.WithCancel(context.Background())

	checker := &mock.QuotaChecker{
		LoadGPUQuotaRequestFn: noLoadRequest,
		CheckGPUQuotaFn: func(_ context.Context, _ string) (*quota.QuotaStatus, error) {
			return &quota.QuotaStatus{
				CurrentVCPUs:  0,
				RequiredVCPUs: 4,
				Sufficient:    false,
				QuotaCode:     "L-DB2BBE81",
			}, nil
		},
		RequestGPUQuotaFn: func(_ context.Context, _ string) (*quota.QuotaRequest, error) {
			requestCalled.Store(true)
			// Cancel context so waitForQuotaApproval exits immediately.
			cancel()
			return &quota.QuotaRequest{
				RequestID: "req-123",
				QuotaCode: "L-DB2BBE81",
				Status:    "PENDING",
			}, nil
		},
	}
	promptFn := func(_ string) string { return "2" }

	result, err := handleGPUQuota(ctx, checker, "g5.xlarge", "us-east-1", promptFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Proceed {
		t.Error("expected Proceed=false when context is cancelled during wait")
	}
	if !requestCalled.Load() {
		t.Error("expected RequestGPUQuota to be called")
	}
}

func TestHandleGPUQuota_ExistingApproved(t *testing.T) {
	var deleted atomic.Bool
	checker := &mock.QuotaChecker{
		LoadGPUQuotaRequestFn: func(_ context.Context, _ string) (*quota.QuotaRequest, error) {
			return &quota.QuotaRequest{
				RequestID: "req-approved",
				QuotaCode: "L-DB2BBE81",
				Status:    "PENDING",
				CreatedAt: time.Now().Add(-time.Hour),
			}, nil
		},
		GetGPUQuotaRequestStatusFn: func(_ context.Context, _ string) (string, error) {
			return "APPROVED", nil
		},
		DeleteGPUQuotaRequestFn: func(_ context.Context, _ string) error {
			deleted.Store(true)
			return nil
		},
	}

	result, err := handleGPUQuota(context.Background(), checker, "g5.xlarge", "us-east-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Proceed {
		t.Error("expected Proceed=true for approved request")
	}
	if !deleted.Load() {
		t.Error("expected DeleteGPUQuotaRequest to be called")
	}
}

func TestHandleGPUQuota_ExistingDenied(t *testing.T) {
	var deleted atomic.Bool
	checker := &mock.QuotaChecker{
		LoadGPUQuotaRequestFn: func(_ context.Context, _ string) (*quota.QuotaRequest, error) {
			return &quota.QuotaRequest{
				RequestID: "req-denied",
				QuotaCode: "L-DB2BBE81",
				Status:    "PENDING",
				CreatedAt: time.Now().Add(-time.Hour),
			}, nil
		},
		GetGPUQuotaRequestStatusFn: func(_ context.Context, _ string) (string, error) {
			return "DENIED", nil
		},
		DeleteGPUQuotaRequestFn: func(_ context.Context, _ string) error {
			deleted.Store(true)
			return nil
		},
	}

	result, err := handleGPUQuota(context.Background(), checker, "g5.xlarge", "us-east-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Proceed {
		t.Error("expected Proceed=false for denied request")
	}
	if !deleted.Load() {
		t.Error("expected DeleteGPUQuotaRequest to be called")
	}
}

func TestHandleGPUQuota_CheckQuotaError(t *testing.T) {
	checker := &mock.QuotaChecker{
		LoadGPUQuotaRequestFn: noLoadRequest,
		CheckGPUQuotaFn: func(_ context.Context, _ string) (*quota.QuotaStatus, error) {
			return nil, errors.New("access denied")
		},
	}

	result, err := handleGPUQuota(context.Background(), checker, "g5.xlarge", "us-east-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Proceed {
		t.Error("expected Proceed=true on graceful degradation when quota check fails")
	}
}
