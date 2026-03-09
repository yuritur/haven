package quota

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Store struct {
	s3Client   *s3.Client
	bucketName string
}

func NewStore(cfg aws.Config, bucketName string) *Store {
	return &Store{
		s3Client:   s3.NewFromConfig(cfg),
		bucketName: bucketName,
	}
}

func (s *Store) key(quotaCode string) string {
	return fmt.Sprintf("quota-requests/%s.json", quotaCode)
}

func (s *Store) Save(ctx context.Context, req QuotaRequest) error {
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return err
	}
	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s.key(req.QuotaCode)),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (s *Store) Load(ctx context.Context, quotaCode string) (*QuotaRequest, error) {
	out, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s.key(quotaCode)),
	})
	if err != nil {
		var nsk *s3types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, nil
		}
		return nil, fmt.Errorf("load quota request %q: %w", quotaCode, err)
	}
	defer out.Body.Close()
	var req QuotaRequest
	if err := json.NewDecoder(out.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *Store) Delete(ctx context.Context, quotaCode string) error {
	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s.key(quotaCode)),
	})
	return err
}
