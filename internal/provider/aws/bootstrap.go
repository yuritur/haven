package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func stateBucketName(accountID string) string {
	return fmt.Sprintf("haven-state-%s", accountID)
}

// ensureStateBucket creates the S3 state bucket if it doesn't exist.
// Safe to call on every deploy — idempotent.
func ensureStateBucket(ctx context.Context, cfg aws.Config, accountID string) (string, error) {
	bucketName := stateBucketName(accountID)
	client := s3.NewFromConfig(cfg)

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	// us-east-1 must NOT have LocationConstraint; all other regions must.
	if cfg.Region != "us-east-1" {
		input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(cfg.Region),
		}
	}

	_, err := client.CreateBucket(ctx, input)
	if err != nil {
		var alreadyOwned *s3types.BucketAlreadyOwnedByYou
		var alreadyExists *s3types.BucketAlreadyExists
		if errors.As(err, &alreadyOwned) || errors.As(err, &alreadyExists) {
			return bucketName, nil
		}
		return "", fmt.Errorf("create state bucket %s: %w", bucketName, err)
	}

	_, err = client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucketName),
		VersioningConfiguration: &s3types.VersioningConfiguration{
			Status: s3types.BucketVersioningStatusEnabled,
		},
	})
	if err != nil {
		return "", fmt.Errorf("enable versioning on %s: %w", bucketName, err)
	}

	on := true
	_, err = client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
		PublicAccessBlockConfiguration: &s3types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       &on,
			BlockPublicPolicy:     &on,
			IgnorePublicAcls:      &on,
			RestrictPublicBuckets: &on,
		},
	})
	if err != nil {
		return "", fmt.Errorf("block public access on %s: %w", bucketName, err)
	}

	return bucketName, nil
}
