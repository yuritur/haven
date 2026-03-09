package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/havenapp/haven/internal/provider/aws/quota"
	"github.com/havenapp/haven/internal/tui"
)

type gpuQuotaChecker interface {
	CheckGPUQuota(ctx context.Context, instanceType string) (*quota.QuotaStatus, error)
	RequestGPUQuota(ctx context.Context, instanceType string) (*quota.QuotaRequest, error)
	LoadGPUQuotaRequest(ctx context.Context, quotaCode string) (*quota.QuotaRequest, error)
	GetGPUQuotaRequestStatus(ctx context.Context, requestID string) (string, error)
	DeleteGPUQuotaRequest(ctx context.Context, quotaCode string) error
}

type gpuQuotaResult struct {
	Proceed bool
}

func handleGPUQuota(ctx context.Context, checker gpuQuotaChecker, instanceType string, region string, promptFn func(string) string) (*gpuQuotaResult, error) {
	quotaCode, err := quota.QuotaCodeForInstance(instanceType)
	if err != nil {
		return nil, err
	}

	// Check for a pending request from a previous run.
	existing, err := checker.LoadGPUQuotaRequest(ctx, quotaCode)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return handleExistingRequest(ctx, checker, existing, promptFn)
	}

	// No pending request — check current quota.
	status, err := checker.CheckGPUQuota(ctx, instanceType)
	if err != nil {
		// Graceful degradation: if we can't check quota (e.g. no IAM permissions),
		// skip the check and let deploy proceed.
		fmt.Fprintf(os.Stderr, "Warning: could not check GPU quota: %v\n", err)
		return &gpuQuotaResult{Proceed: true}, nil
	}

	if status.Sufficient {
		return &gpuQuotaResult{Proceed: true}, nil
	}

	return handleInsufficientQuota(ctx, checker, status, instanceType, region, promptFn)
}

func handleExistingRequest(ctx context.Context, checker gpuQuotaChecker, req *quota.QuotaRequest, promptFn func(string) string) (*gpuQuotaResult, error) {
	status, err := checker.GetGPUQuotaRequestStatus(ctx, req.RequestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not check quota request status: %v\n", err)
		return &gpuQuotaResult{Proceed: true}, nil
	}

	switch status {
	case "APPROVED":
		fmt.Println("GPU quota increase approved!")
		_ = checker.DeleteGPUQuotaRequest(ctx, req.QuotaCode)
		return &gpuQuotaResult{Proceed: true}, nil

	case "DENIED", "NOT_APPLICABLE":
		fmt.Printf("GPU quota increase request was %s.\n", strings.ToLower(status))
		fmt.Println("Please request a quota increase manually and try again.")
		_ = checker.DeleteGPUQuotaRequest(ctx, req.QuotaCode)
		return &gpuQuotaResult{Proceed: false}, nil

	default: // PENDING, CASE_OPENED
		fmt.Printf("A GPU quota increase request is pending (status: %s, submitted: %s).\n",
			status, req.CreatedAt.Format(time.RFC3339))
		choice := promptFn("Wait for approval? [Y/n]: ")
		choice = strings.TrimSpace(strings.ToLower(choice))
		if choice == "n" || choice == "no" {
			fmt.Println("Run `haven deploy` again when the quota is approved.")
			return &gpuQuotaResult{Proceed: false}, nil
		}
		return waitForQuotaApproval(ctx, checker, req.RequestID, req.QuotaCode)
	}
}

func handleInsufficientQuota(ctx context.Context, checker gpuQuotaChecker, status *quota.QuotaStatus, instanceType string, region string, promptFn func(string) string) (*gpuQuotaResult, error) {
	fmt.Printf("\nYour AWS account has %.0f vCPU quota for this instance family (%d required for %s).\n\n",
		status.CurrentVCPUs, status.RequiredVCPUs, instanceType)
	fmt.Println("  [1] I'll request the increase myself")
	fmt.Println("  [2] Let Haven request it (may take several minutes)")

	choice := promptFn("\nChoice [1/2]: ")
	choice = strings.TrimSpace(choice)

	switch choice {
	case "2":
		return submitAndWait(ctx, checker, instanceType, status.QuotaCode)
	default:
		printManualInstructions(status.QuotaCode, status.RequiredVCPUs, region)
		return &gpuQuotaResult{Proceed: false}, nil
	}
}

func printManualInstructions(quotaCode string, requiredVCPUs int, region string) {
	fmt.Printf("\nRequest a quota increase at:\n")
	fmt.Printf("  https://console.aws.amazon.com/servicequotas/home?region=%s#!/services/ec2/quotas/%s\n\n", region, quotaCode)
	fmt.Printf("Or via CLI:\n")
	fmt.Printf("  aws service-quotas request-service-quota-increase \\\n")
	fmt.Printf("    --service-code ec2 --quota-code %s --desired-value %d\n\n", quotaCode, requiredVCPUs)
	fmt.Println("Then run `haven deploy <model>` again.")
}

func submitAndWait(ctx context.Context, checker gpuQuotaChecker, instanceType string, quotaCode string) (*gpuQuotaResult, error) {
	fmt.Println("\nRequesting quota increase...")
	req, err := checker.RequestGPUQuota(ctx, instanceType)
	if err != nil {
		return nil, fmt.Errorf("request quota increase: %w", err)
	}
	fmt.Printf("Request submitted (ID: %s)\n", req.RequestID)
	return waitForQuotaApproval(ctx, checker, req.RequestID, quotaCode)
}

func waitForQuotaApproval(ctx context.Context, checker gpuQuotaChecker, requestID string, quotaCode string) (*gpuQuotaResult, error) {
	spin := tui.StartSpinner("Waiting for quota approval...")
	defer spin.Stop()

	for {
		select {
		case <-ctx.Done():
			spin.Stop()
			fmt.Println("\nInterrupted. The quota request is saved — run `haven deploy` again to check its status.")
			return &gpuQuotaResult{Proceed: false}, nil
		case <-time.After(30 * time.Second):
		}

		status, err := checker.GetGPUQuotaRequestStatus(ctx, requestID)
		if err != nil {
			if ctx.Err() != nil {
				spin.Stop()
				fmt.Println("\nInterrupted. The quota request is saved — run `haven deploy` again to check its status.")
				return &gpuQuotaResult{Proceed: false}, nil
			}
			return nil, fmt.Errorf("poll quota request: %w", err)
		}

		switch status {
		case "APPROVED":
			spin.Stop()
			fmt.Println("GPU quota increase approved!")
			_ = checker.DeleteGPUQuotaRequest(ctx, quotaCode)
			return &gpuQuotaResult{Proceed: true}, nil
		case "DENIED", "NOT_APPLICABLE":
			spin.Stop()
			fmt.Printf("GPU quota increase request was %s.\n", strings.ToLower(status))
			fmt.Println("Please request a quota increase manually and try again.")
			_ = checker.DeleteGPUQuotaRequest(ctx, quotaCode)
			return &gpuQuotaResult{Proceed: false}, nil
		}
	}
}

func stdinPrompt(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text()
	}
	return ""
}
