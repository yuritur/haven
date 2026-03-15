package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/mock"
)

func TestRunCost_Basic(t *testing.T) {
	prov := &mock.Provider{
		LoadDeploymentFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:           "haven-a1b2c3d4",
				Model:        "llama3.2:1b",
				InstanceType: "t3.large",
				InstanceID:   "i-1234567890",
				CreatedAt:    time.Now().Add(-24 * time.Hour),
			}, nil
		},
		EstimateCostFn: func(ctx context.Context, d provider.Deployment) (*provider.CostEstimate, error) {
			return &provider.CostEstimate{
				Total:  2.50,
				Uptime: 24 * time.Hour,
			}, nil
		},
	}

	var buf bytes.Buffer
	err := runCost(context.Background(), prov, "haven-a1b2c3d4", &buf, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "$") {
		t.Errorf("expected output to contain '$', got: %s", out)
	}
	if !strings.Contains(out, "haven-a1b2c3d4") {
		t.Errorf("expected output to contain deployment ID, got: %s", out)
	}
}

func TestRunCost_Projected(t *testing.T) {
	prov := &mock.Provider{
		LoadDeploymentFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:           "haven-a1b2c3d4",
				Model:        "llama3.2:1b",
				InstanceType: "t3.large",
				InstanceID:   "i-1234567890",
				CreatedAt:    time.Now().Add(-24 * time.Hour),
			}, nil
		},
		EstimateCostFn: func(ctx context.Context, d provider.Deployment) (*provider.CostEstimate, error) {
			return &provider.CostEstimate{
				Total:  2.50,
				Uptime: 24 * time.Hour,
			}, nil
		},
		ProjectCostFn: func(ctx context.Context, d provider.Deployment) (*provider.CostEstimate, error) {
			return &provider.CostEstimate{
				Total: 50.00,
			}, nil
		},
	}

	var buf bytes.Buffer
	err := runCost(context.Background(), prov, "haven-a1b2c3d4", &buf, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Projected:") {
		t.Errorf("expected projected output, got: %s", out)
	}
}

func TestRunCost_NotFound(t *testing.T) {
	prov := &mock.Provider{
		LoadDeploymentFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return nil, fmt.Errorf("deployment not found")
		},
	}

	var buf bytes.Buffer
	err := runCost(context.Background(), prov, "nonexistent", &buf, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunCost_NoCostEstimator(t *testing.T) {
	prov := &noCostProvider{
		LoadDeploymentFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:           "haven-a1b2c3d4",
				Model:        "llama3.2:1b",
				InstanceType: "t3.large",
				InstanceID:   "i-1234567890",
				CreatedAt:    time.Now().Add(-24 * time.Hour),
			}, nil
		},
	}

	var buf bytes.Buffer
	err := runCost(context.Background(), prov, "haven-a1b2c3d4", &buf, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "not available") {
		t.Errorf("expected 'not available' message, got: %s", out)
	}
}

// noCostProvider implements Provider but not CostEstimator.
type noCostProvider struct {
	LoadDeploymentFn func(ctx context.Context, id string) (*provider.Deployment, error)
}

func (p *noCostProvider) Identity(context.Context) (provider.Identity, error) {
	return provider.Identity{}, nil
}
func (p *noCostProvider) List(context.Context) ([]provider.Deployment, error) { return nil, nil }
func (p *noCostProvider) LoadDeployment(ctx context.Context, id string) (*provider.Deployment, error) {
	return p.LoadDeploymentFn(ctx, id)
}
func (p *noCostProvider) SaveDeployment(context.Context, provider.Deployment) error { return nil }
func (p *noCostProvider) DeleteDeployment(context.Context, string) error            { return nil }
func (p *noCostProvider) EnsureQuota(context.Context, string, models.Runtime, provider.Prompter) error {
	return nil
}
func (p *noCostProvider) Deploy(context.Context, provider.DeployInput) (provider.DeployResult, error) {
	return provider.DeployResult{}, nil
}
func (p *noCostProvider) Destroy(context.Context, string) error { return nil }
func (p *noCostProvider) Stop(context.Context, string) error    { return nil }
func (p *noCostProvider) Start(context.Context, string) error   { return nil }
