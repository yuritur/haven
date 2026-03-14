package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/mock"
)

func TestRunCert_PEM(t *testing.T) {
	store := &mock.Provider{
		LoadDeploymentFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:      "haven-test1234",
				TLSCert: "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----\n",
			}, nil
		},
	}

	err := runCert(context.Background(), store, "haven-test1234", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestRunCert_Fingerprint(t *testing.T) {
	store := &mock.Provider{
		LoadDeploymentFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:             "haven-test1234",
				TLSFingerprint: "sha256:abcdef1234567890",
			}, nil
		},
	}

	err := runCert(context.Background(), store, "haven-test1234", true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestRunCert_NoCert(t *testing.T) {
	store := &mock.Provider{
		LoadDeploymentFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:      "haven-test1234",
				TLSCert: "",
			}, nil
		},
	}

	err := runCert(context.Background(), store, "haven-test1234", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no TLS certificate") {
		t.Errorf("error %q should contain 'no TLS certificate'", err.Error())
	}
}

func TestRunCert_NoFingerprint(t *testing.T) {
	store := &mock.Provider{
		LoadDeploymentFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return &provider.Deployment{
				ID:             "haven-test1234",
				TLSFingerprint: "",
			}, nil
		},
	}

	err := runCert(context.Background(), store, "haven-test1234", true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no TLS fingerprint") {
		t.Errorf("error %q should contain 'no TLS fingerprint'", err.Error())
	}
}

func TestRunCert_NotFound(t *testing.T) {
	store := &mock.Provider{
		LoadDeploymentFn: func(ctx context.Context, id string) (*provider.Deployment, error) {
			return nil, fmt.Errorf("deployment not found")
		},
	}

	err := runCert(context.Background(), store, "nonexistent", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
