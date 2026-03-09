package aws

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/aws/quota"
	"github.com/havenapp/haven/internal/tui"
)

func (p *AWSProvider) EnsureQuota(ctx context.Context, instanceType string, promptFn func(string) string) error {
	if !models.IsGPUInstance(instanceType) {
		return nil
	}

	quotaCode, err := quota.QuotaCodeForInstance(instanceType)
	if err != nil {
		return err
	}

	existing, err := p.quotaStore.Load(ctx, quotaCode)
	if err != nil {
		return err
	}
	if existing != nil {
		return p.handleExistingQuotaRequest(ctx, existing, promptFn)
	}

	status, err := quota.CheckQuota(ctx, p.cfg, instanceType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not check GPU quota: %v\n", err)
		return nil
	}

	if status.Sufficient {
		return nil
	}

	return p.handleInsufficientQuota(ctx, status, instanceType, promptFn)
}

func (p *AWSProvider) resolveTerminalStatus(ctx context.Context, status string, quotaCode string) (proceed bool, terminal bool) {
	switch status {
	case "APPROVED":
		fmt.Println("GPU quota increase approved!")
		_ = p.quotaStore.Delete(ctx, quotaCode)
		return true, true
	case "DENIED", "CASE_CLOSED":
		fmt.Printf("GPU quota increase request was %s.\n", strings.ToLower(status))
		fmt.Println("Please request a quota increase manually and try again.")
		_ = p.quotaStore.Delete(ctx, quotaCode)
		return false, true
	default:
		return false, false
	}
}

func (p *AWSProvider) handleExistingQuotaRequest(ctx context.Context, req *quota.QuotaRequest, promptFn func(string) string) error {
	status, err := quota.GetRequestStatus(ctx, p.cfg, req.RequestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not check quota request status: %v\n", err)
		return nil
	}

	proceed, terminal := p.resolveTerminalStatus(ctx, status, req.QuotaCode)
	if terminal {
		if !proceed {
			return provider.ErrQuotaUserExit
		}
		return nil
	}

	fmt.Printf("A GPU quota increase request is pending (status: %s, submitted: %s).\n",
		status, req.CreatedAt.Format(time.RFC3339))
	choice := promptFn("\033[33mWait for approval? [Y/n]:\033[0m ")
	choice = strings.TrimSpace(strings.ToLower(choice))
	if choice == "n" || choice == "no" {
		fmt.Println("Run `haven deploy` again when the quota is approved.")
		return provider.ErrQuotaUserExit
	}
	return p.waitForQuotaApproval(ctx, req.RequestID, req.QuotaCode)
}

func (p *AWSProvider) handleInsufficientQuota(ctx context.Context, status *quota.QuotaStatus, instanceType string, promptFn func(string) string) error {
	fmt.Printf("\nYour AWS account has %.0f vCPU quota for this instance family (%d required for %s).\n",
		status.CurrentVCPUs, status.RequiredVCPUs, instanceType)

	if !status.APIAvailable {
		fmt.Println("Quota is not yet registered in AWS Service Quotas for this account.")
		p.printManualInstructions(status.QuotaCode, status.RequiredVCPUs)
		return provider.ErrQuotaUserExit
	}

	fmt.Println()
	fmt.Println("  [1] I'll request the increase myself")
	fmt.Println("  [2] Let Haven request it (may take several minutes)")

	choice := promptFn("\n\033[33mChoice [1/2]:\033[0m ")
	choice = strings.TrimSpace(choice)

	switch choice {
	case "2":
		return p.submitAndWait(ctx, instanceType, status.QuotaCode)
	default:
		p.printManualInstructions(status.QuotaCode, status.RequiredVCPUs)
		return provider.ErrQuotaUserExit
	}
}

func (p *AWSProvider) printManualInstructions(quotaCode string, requiredVCPUs int) {
	fmt.Printf("\nRequest a quota increase at:\n")
	fmt.Printf("  https://console.aws.amazon.com/servicequotas/home?region=%s#!/services/ec2/quotas/%s\n\n", p.identity.Region, quotaCode)
	fmt.Printf("Or via CLI:\n")
	fmt.Printf("  aws service-quotas request-service-quota-increase \\\n")
	fmt.Printf("    --service-code ec2 --quota-code %s --desired-value %d\n\n", quotaCode, requiredVCPUs)
	fmt.Println("Then run `haven deploy <model>` again.")
}

func (p *AWSProvider) submitAndWait(ctx context.Context, instanceType string, quotaCode string) error {
	fmt.Println("\nRequesting quota increase...")

	vcpus, err := quota.VCPUsForInstance(instanceType)
	if err != nil {
		return err
	}
	req, err := quota.RequestIncrease(ctx, p.cfg, quotaCode, float64(vcpus))
	if err != nil {
		return fmt.Errorf("request quota increase: %w", err)
	}
	req.InstanceType = instanceType
	if err := p.quotaStore.Save(ctx, *req); err != nil {
		return fmt.Errorf("save quota request: %w", err)
	}

	fmt.Printf("Request submitted (ID: %s)\n", req.RequestID)
	return p.waitForQuotaApproval(ctx, req.RequestID, quotaCode)
}

func (p *AWSProvider) waitForQuotaApproval(ctx context.Context, requestID string, quotaCode string) error {
	spin := tui.StartSpinner("Waiting for quota approval...")
	defer spin.Stop()

	for {
		select {
		case <-ctx.Done():
			spin.Stop()
			fmt.Println("\nInterrupted. The quota request is saved — run `haven deploy` again to check its status.")
			return provider.ErrQuotaUserExit
		case <-time.After(30 * time.Second):
		}

		status, err := quota.GetRequestStatus(ctx, p.cfg, requestID)
		if err != nil {
			if ctx.Err() != nil {
				spin.Stop()
				fmt.Println("\nInterrupted. The quota request is saved — run `haven deploy` again to check its status.")
				return provider.ErrQuotaUserExit
			}
			return fmt.Errorf("poll quota request: %w", err)
		}

		proceed, terminal := p.resolveTerminalStatus(ctx, status, quotaCode)
		if terminal {
			spin.Stop()
			if !proceed {
				return provider.ErrQuotaUserExit
			}
			return nil
		}
	}
}
