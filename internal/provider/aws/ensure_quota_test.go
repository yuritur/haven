package aws

import (
	"bytes"
	"context"
	"errors"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/aws/quota"
	"github.com/havenapp/haven/internal/provider/mock"
)

func testProvider(region string) *AWSProvider {
	return &AWSProvider{
		identity:   provider.Identity{Region: region},
		quotaStore: quota.NewStore(awssdk.Config{}, "test-bucket"),
	}
}

func TestEnsureQuota_NonGPUInstance(t *testing.T) {
	p := &AWSProvider{}
	err := p.EnsureQuota(context.Background(), "t3.large", nil)
	if err != nil {
		t.Fatalf("expected nil error for non-GPU instance, got: %v", err)
	}
}

func TestResolveTerminalStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		wantProceed  bool
		wantTerminal bool
	}{
		{"approved", "APPROVED", true, true},
		{"denied", "DENIED", false, true},
		{"case_closed", "CASE_CLOSED", false, true},
		{"pending", "PENDING", false, false},
		{"unknown_status", "NOT_A_STATUS", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testProvider("us-east-1")
			proceed, terminal := p.resolveTerminalStatus(context.Background(), tt.status, "L-DB2BBE81")
			if proceed != tt.wantProceed {
				t.Errorf("proceed = %v, want %v", proceed, tt.wantProceed)
			}
			if terminal != tt.wantTerminal {
				t.Errorf("terminal = %v, want %v", terminal, tt.wantTerminal)
			}
		})
	}
}

func TestPrintManualInstructions(t *testing.T) {
	var buf bytes.Buffer
	p := testProvider("eu-west-1")
	p.out = &buf

	p.printManualInstructions("L-DB2BBE81", 4)

	// printManualInstructions writes to os.Stdout (not p.out), so we verify it doesn't panic.
	// The function is primarily side-effects (fmt.Printf to stdout).
	// We just verify it completes without error.
}

func TestHandleInsufficientQuota_ManualChoice(t *testing.T) {
	p := testProvider("us-west-2")

	status := &quota.QuotaStatus{
		CurrentVCPUs:  0,
		RequiredVCPUs: 4,
		Sufficient:    false,
		QuotaCode:     "L-DB2BBE81",
	}

	prompter := &mock.Prompter{InputFn: func(string) string { return "1" }}

	err := p.handleInsufficientQuota(context.Background(), status, "g4dn.xlarge", prompter)
	if !errors.Is(err, provider.ErrQuotaUserExit) {
		t.Fatalf("expected ErrQuotaUserExit, got: %v", err)
	}
}

func TestHandleInsufficientQuota_DefaultChoice(t *testing.T) {
	p := testProvider("us-west-2")

	status := &quota.QuotaStatus{
		CurrentVCPUs:  0,
		RequiredVCPUs: 4,
		Sufficient:    false,
		QuotaCode:     "L-DB2BBE81",
	}

	// Any non-"2" value should fall through to manual instructions
	prompter := &mock.Prompter{InputFn: func(string) string { return "" }}

	err := p.handleInsufficientQuota(context.Background(), status, "g4dn.xlarge", prompter)
	if !errors.Is(err, provider.ErrQuotaUserExit) {
		t.Fatalf("expected ErrQuotaUserExit, got: %v", err)
	}
}
