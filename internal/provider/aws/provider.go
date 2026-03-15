package aws

import (
	"context"
	"io"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/aws/cfn"
	"github.com/havenapp/haven/internal/provider/aws/quota"
)

type AWSProvider struct {
	cfg        awssdk.Config
	identity   provider.Identity
	bucketName string
	stateStore *S3StateStore
	quotaStore *quota.Store
	out        io.Writer
}

var _ provider.Provider = (*AWSProvider)(nil)

func Build(ctx context.Context, out io.Writer) (provider.Provider, error) {
	ar, err := authenticate(ctx, out)
	if err != nil {
		return nil, err
	}

	store, err := newS3StateStore(ctx, ar.cfg, ar.identity.AccountID)
	if err != nil {
		return nil, err
	}

	p := &AWSProvider{
		cfg:        ar.cfg,
		out:        out,
		bucketName: store.bucketName,
		stateStore: store,
		quotaStore: quota.NewStore(ar.cfg, store.bucketName),
		identity:   ar.identity,
	}

	return p, nil
}

func (p *AWSProvider) Identity(_ context.Context) (provider.Identity, error) {
	return p.identity, nil
}

func (p *AWSProvider) List(ctx context.Context) ([]provider.Deployment, error) {
	return p.stateStore.List(ctx)
}

func (p *AWSProvider) LoadDeployment(ctx context.Context, id string) (*provider.Deployment, error) {
	return p.stateStore.Load(ctx, id)
}

func (p *AWSProvider) SaveDeployment(ctx context.Context, d provider.Deployment) error {
	return p.stateStore.Save(ctx, d)
}

func (p *AWSProvider) DeleteDeployment(ctx context.Context, id string) error {
	return p.stateStore.Delete(ctx, id)
}

func (p *AWSProvider) Deploy(ctx context.Context, input provider.DeployInput) (provider.DeployResult, error) {
	spec, err := ResolveInstance(input.Model, input.Runtime)
	if err != nil {
		return provider.DeployResult{}, err
	}

	result, err := cfn.Deploy(ctx, p.cfg, cfn.DeployInput{
		StackName:    input.DeploymentID,
		Runtime:      input.Runtime,
		ModelTag:     input.ModelTag,
		InstanceType: spec.InstanceType,
		UserIP:       input.UserIP,
		APIKey:       input.APIKey,
		TLSCert:      input.TLSCert,
		TLSKey:       input.TLSKey,
		EBSVolumeGB:  spec.EBSVolumeGB,
		HFRepo:       input.HFRepo,
		HFFile:       input.HFFile,
		GPU:          spec.GPU,
		Out:          p.out,
	})
	if err != nil {
		return provider.DeployResult{}, err
	}
	return provider.DeployResult{
		ProviderRef:  result.StackName,
		InstanceID:   result.InstanceID,
		PublicIP:     result.PublicIP,
		InstanceType: spec.InstanceType,
		GPU:          spec.GPU,
	}, nil
}

func (p *AWSProvider) Destroy(ctx context.Context, providerRef string) error {
	return cfn.Destroy(ctx, p.cfg, providerRef, p.out)
}

func (p *AWSProvider) Stop(ctx context.Context, instanceID string) error {
	client := ec2.NewFromConfig(p.cfg)
	_, err := client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	return err
}

func (p *AWSProvider) Start(ctx context.Context, instanceID string) error {
	client := ec2.NewFromConfig(p.cfg)
	_, err := client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	return err
}
