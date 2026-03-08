package cli

import (
	"context"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"github.com/havenapp/haven/internal/provider/mock"
)

func TestGenerateAPIKey(t *testing.T) {
	key, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey() returned error: %v", err)
	}
	if !strings.HasPrefix(key, "sk-haven-") {
		t.Errorf("key %q does not start with sk-haven-", key)
	}
	// "sk-haven-" (9) + 36 hex chars = 45
	if len(key) != 45 {
		t.Errorf("key length = %d, want 45", len(key))
	}
	suffix := key[len("sk-haven-"):]
	if _, err := hex.DecodeString(suffix); err != nil {
		t.Errorf("suffix %q is not valid hex: %v", suffix, err)
	}

	key2, err := generateAPIKey()
	if err != nil {
		t.Fatalf("second generateAPIKey() returned error: %v", err)
	}
	if key == key2 {
		t.Error("two calls produced identical keys")
	}
}

func TestGenerateDeploymentID(t *testing.T) {
	id, err := generateDeploymentID()
	if err != nil {
		t.Fatalf("generateDeploymentID() returned error: %v", err)
	}
	if !strings.HasPrefix(id, "haven-") {
		t.Errorf("id %q does not start with haven-", id)
	}
	if len(id) != 14 {
		t.Errorf("id length = %d, want 14", len(id))
	}
	suffix := id[len("haven-"):]
	if _, err := hex.DecodeString(suffix); err != nil {
		t.Errorf("suffix %q is not valid hex: %v", suffix, err)
	}

	id2, err := generateDeploymentID()
	if err != nil {
		t.Fatalf("second generateDeploymentID() returned error: %v", err)
	}
	if id == id2 {
		t.Error("two calls produced identical IDs")
	}
}

func TestRunDeploy_UnknownModel(t *testing.T) {
	err := runDeploy(context.Background(), &mock.Provider{}, &mock.StateStore{}, "aws", "nonexistent:model", true, io.Discard)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown model") {
		t.Errorf("error %q should contain 'unknown model'", err.Error())
	}
}
