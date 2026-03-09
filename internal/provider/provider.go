package provider

import (
	"context"
	"errors"
	"time"

	"github.com/havenapp/haven/internal/models"
)

var ErrQuotaUserExit = errors.New("quota: user chose manual resolution")
var ErrNoAccount = errors.New("no AWS account")

type Prompter interface {
	Confirm(message string) bool
	Input(prompt string) string
	Secret(prompt string) string
	Select(prompt string, options []string) int
	Print(message string)
}

type Identity struct {
	AccountID string
	ARN       string
	Region    string
}

type DeployInput struct {
	DeploymentID   string
	Runtime        models.Runtime
	ModelTag       string
	InstanceType   string
	UserIP         string
	APIKey         string
	TLSCert        string
	TLSKey         string
	TLSFingerprint string
	EBSVolumeGB    int
}

type DeployResult struct {
	ProviderRef string
	InstanceID  string
	PublicIP    string
}

type Deployment struct {
	ID           string    `json:"deployment_id"`
	Provider     string    `json:"provider"`
	ProviderRef  string    `json:"provider_ref"`
	CreatedAt    time.Time `json:"created_at"`
	Region       string    `json:"region"`
	Model        string    `json:"model"`
	InstanceType string    `json:"instance_type"`
	InstanceID   string    `json:"instance_id"`
	PublicIP     string    `json:"public_ip"`
	Endpoint     string    `json:"endpoint"`
	APIKey       string    `json:"api_key"`
	// TLSKey is intentionally absent — private key is never persisted to state.
	TLSCert        string `json:"tls_cert"`
	TLSFingerprint string `json:"tls_fingerprint"`
}

type Provider interface {
	Identity(ctx context.Context) (Identity, error)
	Deploy(ctx context.Context, input DeployInput) (DeployResult, error)
	Destroy(ctx context.Context, providerRef string) error
}

type StateStore interface {
	Save(ctx context.Context, d Deployment) error
	Load(ctx context.Context, id string) (*Deployment, error)
	List(ctx context.Context) ([]Deployment, error)
	Delete(ctx context.Context, id string) error
}
