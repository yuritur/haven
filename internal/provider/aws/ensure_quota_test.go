package aws

import (
	"context"
	"testing"
)

func TestEnsureQuota_NonGPUInstance(t *testing.T) {
	p := &AWSProvider{}
	err := p.EnsureQuota(context.Background(), "t3.large", nil)
	if err != nil {
		t.Fatalf("expected nil error for non-GPU instance, got: %v", err)
	}
}
