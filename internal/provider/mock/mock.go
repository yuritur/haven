package mock

import (
	"context"
	"errors"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/aws/quota"
)

var _ provider.Provider = (*Provider)(nil)
var _ provider.StateStore = (*StateStore)(nil)

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

type QuotaChecker struct {
	CheckGPUQuotaFn            func(ctx context.Context, instanceType string) (*quota.QuotaStatus, error)
	RequestGPUQuotaFn          func(ctx context.Context, instanceType string) (*quota.QuotaRequest, error)
	LoadGPUQuotaRequestFn      func(ctx context.Context, quotaCode string) (*quota.QuotaRequest, error)
	GetGPUQuotaRequestStatusFn func(ctx context.Context, requestID string) (string, error)
	DeleteGPUQuotaRequestFn    func(ctx context.Context, quotaCode string) error
}

func (m *QuotaChecker) CheckGPUQuota(ctx context.Context, instanceType string) (*quota.QuotaStatus, error) {
	if m.CheckGPUQuotaFn == nil {
		return nil, errors.New("mock: CheckGPUQuotaFn not configured")
	}
	return m.CheckGPUQuotaFn(ctx, instanceType)
}

func (m *QuotaChecker) RequestGPUQuota(ctx context.Context, instanceType string) (*quota.QuotaRequest, error) {
	if m.RequestGPUQuotaFn == nil {
		return nil, errors.New("mock: RequestGPUQuotaFn not configured")
	}
	return m.RequestGPUQuotaFn(ctx, instanceType)
}

func (m *QuotaChecker) LoadGPUQuotaRequest(ctx context.Context, quotaCode string) (*quota.QuotaRequest, error) {
	if m.LoadGPUQuotaRequestFn == nil {
		return nil, errors.New("mock: LoadGPUQuotaRequestFn not configured")
	}
	return m.LoadGPUQuotaRequestFn(ctx, quotaCode)
}

func (m *QuotaChecker) GetGPUQuotaRequestStatus(ctx context.Context, requestID string) (string, error) {
	if m.GetGPUQuotaRequestStatusFn == nil {
		return "", errors.New("mock: GetGPUQuotaRequestStatusFn not configured")
	}
	return m.GetGPUQuotaRequestStatusFn(ctx, requestID)
}

func (m *QuotaChecker) DeleteGPUQuotaRequest(ctx context.Context, quotaCode string) error {
	if m.DeleteGPUQuotaRequestFn == nil {
		return errors.New("mock: DeleteGPUQuotaRequestFn not configured")
	}
	return m.DeleteGPUQuotaRequestFn(ctx, quotaCode)
}
