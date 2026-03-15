package mock

import (
	"context"
	"errors"

	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/provider"
)

var _ provider.Provider = (*Provider)(nil)
var _ provider.CostEstimator = (*Provider)(nil)
var _ provider.Prompter = (*Prompter)(nil)

type Provider struct {
	IdentityFn         func(ctx context.Context) (provider.Identity, error)
	ListFn             func(ctx context.Context) ([]provider.Deployment, error)
	LoadDeploymentFn   func(ctx context.Context, id string) (*provider.Deployment, error)
	SaveDeploymentFn   func(ctx context.Context, d provider.Deployment) error
	DeleteDeploymentFn func(ctx context.Context, id string) error
	EnsureQuotaFn      func(ctx context.Context, model string, runtime models.RuntimeName, prompter provider.Prompter) error
	DeployFn           func(ctx context.Context, input provider.DeployInput) (provider.DeployResult, error)
	DestroyFn          func(ctx context.Context, providerRef string) error
	StopFn             func(ctx context.Context, instanceID string) error
	StartFn            func(ctx context.Context, instanceID string) error
	EstimateCostFn     func(ctx context.Context, d provider.Deployment) (*provider.CostEstimate, error)
	ProjectCostFn      func(ctx context.Context, d provider.Deployment) (*provider.CostEstimate, error)
}

func (m *Provider) Identity(ctx context.Context) (provider.Identity, error) {
	if m.IdentityFn == nil {
		return provider.Identity{}, errors.New("mock: IdentityFn not configured")
	}
	return m.IdentityFn(ctx)
}

func (m *Provider) EnsureQuota(ctx context.Context, model string, runtime models.RuntimeName, prompter provider.Prompter) error {
	if m.EnsureQuotaFn == nil {
		return nil
	}
	return m.EnsureQuotaFn(ctx, model, runtime, prompter)
}

func (m *Provider) Deploy(ctx context.Context, input provider.DeployInput) (provider.DeployResult, error) {
	if m.DeployFn == nil {
		return provider.DeployResult{}, errors.New("mock: DeployFn not configured")
	}
	return m.DeployFn(ctx, input)
}

func (m *Provider) Destroy(ctx context.Context, providerRef string) error {
	if m.DestroyFn == nil {
		return errors.New("mock: DestroyFn not configured")
	}
	return m.DestroyFn(ctx, providerRef)
}

func (m *Provider) Stop(ctx context.Context, instanceID string) error {
	if m.StopFn == nil {
		return errors.New("mock: StopFn not configured")
	}
	return m.StopFn(ctx, instanceID)
}

func (m *Provider) Start(ctx context.Context, instanceID string) error {
	if m.StartFn == nil {
		return errors.New("mock: StartFn not configured")
	}
	return m.StartFn(ctx, instanceID)
}

func (m *Provider) EstimateCost(ctx context.Context, d provider.Deployment) (*provider.CostEstimate, error) {
	if m.EstimateCostFn == nil {
		return nil, errors.New("mock: EstimateCostFn not configured")
	}
	return m.EstimateCostFn(ctx, d)
}

func (m *Provider) ProjectCost(ctx context.Context, d provider.Deployment) (*provider.CostEstimate, error) {
	if m.ProjectCostFn == nil {
		return nil, errors.New("mock: ProjectCostFn not configured")
	}
	return m.ProjectCostFn(ctx, d)
}

func (m *Provider) List(ctx context.Context) ([]provider.Deployment, error) {
	if m.ListFn == nil {
		return nil, errors.New("mock: ListFn not configured")
	}
	return m.ListFn(ctx)
}

func (m *Provider) LoadDeployment(ctx context.Context, id string) (*provider.Deployment, error) {
	if m.LoadDeploymentFn == nil {
		return nil, errors.New("mock: LoadDeploymentFn not configured")
	}
	return m.LoadDeploymentFn(ctx, id)
}

func (m *Provider) SaveDeployment(ctx context.Context, d provider.Deployment) error {
	if m.SaveDeploymentFn == nil {
		return errors.New("mock: SaveDeploymentFn not configured")
	}
	return m.SaveDeploymentFn(ctx, d)
}

func (m *Provider) DeleteDeployment(ctx context.Context, id string) error {
	if m.DeleteDeploymentFn == nil {
		return errors.New("mock: DeleteDeploymentFn not configured")
	}
	return m.DeleteDeploymentFn(ctx, id)
}

type Prompter struct {
	ConfirmFn func(string) bool
	InputFn   func(string) string
	SecretFn  func(string) string
	SelectFn  func(string, []string) int
	PrintFn   func(string)
}

func (m *Prompter) Confirm(message string) bool {
	if m.ConfirmFn == nil {
		return false
	}
	return m.ConfirmFn(message)
}

func (m *Prompter) Input(prompt string) string {
	if m.InputFn == nil {
		return ""
	}
	return m.InputFn(prompt)
}

func (m *Prompter) Secret(prompt string) string {
	if m.SecretFn == nil {
		return ""
	}
	return m.SecretFn(prompt)
}

func (m *Prompter) Select(prompt string, options []string) int {
	if m.SelectFn == nil {
		return -1
	}
	return m.SelectFn(prompt, options)
}

func (m *Prompter) Print(message string) {
	if m.PrintFn == nil {
		return
	}
	m.PrintFn(message)
}
