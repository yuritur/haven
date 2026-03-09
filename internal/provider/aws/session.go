package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/havenapp/haven/internal/provider"
)

type Session struct {
	Provider  string `json:"provider"`
	Profile   string `json:"profile"`
	AccountID string `json:"account_id"`
	Region    string `json:"region"`
}

func sessionPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".haven", "session.json"), nil
}

func SaveSession(s Session) error {
	p, err := sessionPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("create ~/.haven: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	return os.WriteFile(p, data, 0600)
}

func LoadSession() (*Session, error) {
	p, err := sessionPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	return &s, nil
}

func ResumeSession(ctx context.Context, out io.Writer) (provider.Provider, provider.StateStore, error) {
	sess, err := LoadSession()
	if err != nil {
		return nil, nil, fmt.Errorf("not logged in. Run: haven login")
	}

	var ar *authResult
	if sess.Profile == "" {
		ar, err = resolveDefault(ctx)
	} else {
		ar, err = resolveProfile(ctx, sess.Profile)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("session expired or invalid. Run: haven login")
	}

	if ar.identity.AccountID != sess.AccountID {
		return nil, nil, fmt.Errorf("session account mismatch (expected %s, got %s). Run: haven login", sess.AccountID, ar.identity.AccountID)
	}

	return initFromResult(ctx, ar, out)
}
