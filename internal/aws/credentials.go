package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type Identity struct {
	AccountID string
	Region    string
}

func LoadConfig(ctx context.Context) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load AWS config: %w", err)
	}
	return cfg, nil
}

func GetIdentity(ctx context.Context, cfg aws.Config) (Identity, error) {
	client := sts.NewFromConfig(cfg)
	out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return Identity{}, fmt.Errorf("GetCallerIdentity failed — check AWS credentials: %w", err)
	}
	return Identity{
		AccountID: aws.ToString(out.Account),
		Region:    cfg.Region,
	}, nil
}
