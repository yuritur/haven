package aws

import (
	"context"
	"fmt"
	"io"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/aws/cfn"
	"github.com/havenapp/haven/internal/provider/aws/quota"
)

type AWSProvider struct {
	cfg        awssdk.Config
	identity   provider.Identity
	bucketName string
	quotaStore *quota.Store
	out        io.Writer
}

var _ provider.Provider = (*AWSProvider)(nil)

func New(ctx context.Context, out io.Writer) (provider.Provider, provider.StateStore, error) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	id, err := getIdentity(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	store, err := newS3StateStore(ctx, cfg, id.AccountID)
	if err != nil {
		return nil, nil, err
	}

	p := &AWSProvider{
		cfg:        cfg,
		out:        out,
		bucketName: store.bucketName,
		quotaStore: quota.NewStore(cfg, store.bucketName),
		identity: provider.Identity{
			AccountID: id.AccountID,
			Region:    id.Region,
		},
	}
	return p, store, nil
}

func (p *AWSProvider) Identity(_ context.Context) (provider.Identity, error) {
	return p.identity, nil
}

func (p *AWSProvider) Deploy(ctx context.Context, input provider.DeployInput) (provider.DeployResult, error) {
	result, err := cfn.Deploy(ctx, p.cfg, cfn.DeployInput{
		StackName:    input.DeploymentID,
		Runtime:      input.Runtime,
		ModelTag:     input.ModelTag,
		InstanceType: input.InstanceType,
		UserIP:       input.UserIP,
		APIKey:       input.APIKey,
		TLSCert:      input.TLSCert,
		TLSKey:       input.TLSKey,
		EBSVolumeGB:  input.EBSVolumeGB,
		Out:          p.out,
	})
	if err != nil {
		return provider.DeployResult{}, err
	}
	return provider.DeployResult{
		ProviderRef: result.StackName,
		InstanceID:  result.InstanceID,
		PublicIP:    result.PublicIP,
	}, nil
}

func (p *AWSProvider) Destroy(ctx context.Context, providerRef string) error {
	return cfn.Destroy(ctx, p.cfg, providerRef, p.out)
}

func (p *AWSProvider) CheckGPUQuota(ctx context.Context, instanceType string) (*quota.QuotaStatus, error) {
	return quota.CheckQuota(ctx, p.cfg, instanceType)
}

func (p *AWSProvider) RequestGPUQuota(ctx context.Context, instanceType string) (*quota.QuotaRequest, error) {
	quotaCode, err := quota.QuotaCodeForInstance(instanceType)
	if err != nil {
		return nil, err
	}
	vcpus, err := quota.VCPUsForInstance(instanceType)
	if err != nil {
		return nil, err
	}
	req, err := quota.RequestIncrease(ctx, p.cfg, quotaCode, float64(vcpus))
	if err != nil {
		return nil, err
	}
	req.InstanceType = instanceType
	if err := p.quotaStore.Save(ctx, *req); err != nil {
		return nil, fmt.Errorf("save quota request: %w", err)
	}
	return req, nil
}

func (p *AWSProvider) LoadGPUQuotaRequest(ctx context.Context, quotaCode string) (*quota.QuotaRequest, error) {
	return p.quotaStore.Load(ctx, quotaCode)
}

func (p *AWSProvider) GetGPUQuotaRequestStatus(ctx context.Context, requestID string) (string, error) {
	return quota.GetRequestStatus(ctx, p.cfg, requestID)
}

func (p *AWSProvider) DeleteGPUQuotaRequest(ctx context.Context, quotaCode string) error {
	return p.quotaStore.Delete(ctx, quotaCode)
}
