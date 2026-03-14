package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/havenapp/haven/internal/provider"
)

type S3StateStore struct {
	s3Client   *s3.Client
	bucketName string
}

func newS3StateStore(ctx context.Context, cfg awssdk.Config, accountID string) (*S3StateStore, error) {
	bucketName, err := ensureStateBucket(ctx, cfg, accountID)
	if err != nil {
		return nil, err
	}
	return &S3StateStore{
		s3Client:   s3.NewFromConfig(cfg),
		bucketName: bucketName,
	}, nil
}

func (s *S3StateStore) key(id string) string {
	return fmt.Sprintf("deployments/%s.json", id)
}

func (s *S3StateStore) Save(ctx context.Context, d provider.Deployment) error {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: awssdk.String(s.bucketName),
		Key:    awssdk.String(s.key(d.ID)),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (s *S3StateStore) Load(ctx context.Context, id string) (*provider.Deployment, error) {
	out, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: awssdk.String(s.bucketName),
		Key:    awssdk.String(s.key(id)),
	})
	if err != nil {
		return nil, fmt.Errorf("deployment %q not found: %w", id, err)
	}
	defer out.Body.Close()
	var d provider.Deployment
	if err := json.NewDecoder(out.Body).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *S3StateStore) List(ctx context.Context) ([]provider.Deployment, error) {
	out, err := s.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: awssdk.String(s.bucketName),
		Prefix: awssdk.String("deployments/"),
	})
	if err != nil {
		return nil, err
	}
	var deployments []provider.Deployment
	var loadErr error
	for _, obj := range out.Contents {
		id := strings.TrimSuffix(strings.TrimPrefix(awssdk.ToString(obj.Key), "deployments/"), ".json")
		d, err := s.Load(ctx, id)
		if err != nil {
			loadErr = errors.Join(loadErr, fmt.Errorf("load deployment %q: %w", id, err))
			continue
		}
		deployments = append(deployments, *d)
	}
	if loadErr != nil {
		return deployments, loadErr
	}
	return deployments, nil
}

func (s *S3StateStore) Delete(ctx context.Context, id string) error {
	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: awssdk.String(s.bucketName),
		Key:    awssdk.String(s.key(id)),
	})
	return err
}
