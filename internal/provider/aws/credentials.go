package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type identity struct {
	AccountID string
	ARN       string
	Region    string
}

func loadConfig(ctx context.Context) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load AWS config: %w", err)
	}
	return cfg, nil
}

func getIdentity(ctx context.Context, cfg aws.Config) (identity, error) {
	client := sts.NewFromConfig(cfg)
	out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return identity{}, fmt.Errorf("GetCallerIdentity failed — check AWS credentials: %w", err)
	}
	return identity{
		AccountID: aws.ToString(out.Account),
		ARN:       aws.ToString(out.Arn),
		Region:    cfg.Region,
	}, nil
}
