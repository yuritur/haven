package state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	havnaws "github.com/havenapp/haven/internal/aws"
)

type Deployment struct {
	ID           string    `json:"deployment_id"`
	CreatedAt    time.Time `json:"created_at"`
	Region       string    `json:"region"`
	StackName    string    `json:"stack_name"`
	Model        string    `json:"model"`
	InstanceType string    `json:"instance_type"`
	InstanceID   string    `json:"instance_id"`
	EIP          string    `json:"eip"`
	Endpoint     string    `json:"endpoint"`
	APIKey       string    `json:"api_key"`
}

type Manager struct {
	s3Client   *s3.Client
	bucketName string
}

func NewManager(ctx context.Context, cfg aws.Config) (*Manager, error) {
	identity, err := havnaws.GetIdentity(ctx, cfg)
	if err != nil {
		return nil, err
	}
	bucketName, err := havnaws.EnsureStateBucket(ctx, cfg, identity.AccountID)
	if err != nil {
		return nil, err
	}
	return &Manager{
		s3Client:   s3.NewFromConfig(cfg),
		bucketName: bucketName,
	}, nil
}

func (m *Manager) key(deploymentID string) string {
	return fmt.Sprintf("deployments/%s.json", deploymentID)
}

func (m *Manager) Save(ctx context.Context, d Deployment) error {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	_, err = m.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(m.bucketName),
		Key:    aws.String(m.key(d.ID)),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (m *Manager) Load(ctx context.Context, deploymentID string) (*Deployment, error) {
	out, err := m.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(m.bucketName),
		Key:    aws.String(m.key(deploymentID)),
	})
	if err != nil {
		return nil, fmt.Errorf("deployment %q not found: %w", deploymentID, err)
	}
	defer out.Body.Close()
	var d Deployment
	if err := json.NewDecoder(out.Body).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (m *Manager) List(ctx context.Context) ([]Deployment, error) {
	out, err := m.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(m.bucketName),
		Prefix: aws.String("deployments/"),
	})
	if err != nil {
		return nil, err
	}
	var deployments []Deployment
	for _, obj := range out.Contents {
		id := strings.TrimSuffix(strings.TrimPrefix(aws.ToString(obj.Key), "deployments/"), ".json")
		d, err := m.Load(ctx, id)
		if err != nil {
			continue
		}
		deployments = append(deployments, *d)
	}
	return deployments, nil
}

func (m *Manager) Delete(ctx context.Context, deploymentID string) error {
	_, err := m.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(m.bucketName),
		Key:    aws.String(m.key(deploymentID)),
	})
	return err
}
