package mock

import (
	"context"
	"errors"

	"github.com/havenapp/haven/internal/provider"
)

var _ provider.Provider = (*Provider)(nil)
var _ provider.StateStore = (*StateStore)(nil)
var _ provider.Prompter = (*Prompter)(nil)

type Provider struct {
	IdentityFn func(ctx context.Context) (provider.Identity, error)
	DeployFn   func(ctx context.Context, input provider.DeployInput) (provider.DeployResult, error)
	DestroyFn  func(ctx context.Context, providerRef string) error
}

func (m *Provider) Identity(ctx context.Context) (provider.Identity, error) {
	if m.IdentityFn == nil {
		return provider.Identity{}, errors.New("mock: IdentityFn not configured")
	}
	return m.IdentityFn(ctx)
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

type StateStore struct {
	SaveFn   func(ctx context.Context, d provider.Deployment) error
	LoadFn   func(ctx context.Context, id string) (*provider.Deployment, error)
	ListFn   func(ctx context.Context) ([]provider.Deployment, error)
	DeleteFn func(ctx context.Context, id string) error
}

func (m *StateStore) Save(ctx context.Context, d provider.Deployment) error {
	if m.SaveFn == nil {
		return errors.New("mock: SaveFn not configured")
	}
	return m.SaveFn(ctx, d)
}

func (m *StateStore) Load(ctx context.Context, id string) (*provider.Deployment, error) {
	if m.LoadFn == nil {
		return nil, errors.New("mock: LoadFn not configured")
	}
	return m.LoadFn(ctx, id)
}

func (m *StateStore) List(ctx context.Context) ([]provider.Deployment, error) {
	if m.ListFn == nil {
		return nil, errors.New("mock: ListFn not configured")
	}
	return m.ListFn(ctx)
}

func (m *StateStore) Delete(ctx context.Context, id string) error {
	if m.DeleteFn == nil {
		return errors.New("mock: DeleteFn not configured")
	}
	return m.DeleteFn(ctx, id)
}

type QuotaEnsurer struct {
	EnsureQuotaFn func(ctx context.Context, instanceType string, promptFn func(string) string) error
}

func (m *QuotaEnsurer) EnsureQuota(ctx context.Context, instanceType string, promptFn func(string) string) error {
	if m.EnsureQuotaFn == nil {
		return errors.New("mock: EnsureQuotaFn not configured")
	}
	return m.EnsureQuotaFn(ctx, instanceType, promptFn)
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
