package aws

import (
	"context"
	"io"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/aws/cfn"
)

type AWSProvider struct {
	cfg      awssdk.Config
	identity provider.Identity
	out      io.Writer
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

	return &AWSProvider{
		cfg: cfg,
		out: out,
		identity: provider.Identity{
			AccountID: id.AccountID,
			Region:    id.Region,
		},
	}, store, nil
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
