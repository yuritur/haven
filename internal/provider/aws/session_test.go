package aws

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadSession(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, home string)
		session Session
		wantErr bool
	}{
		{
			name: "round-trip preserves all fields",
			session: Session{
				Provider:  "aws",
				Profile:   "haven",
				AccountID: "123456789012",
				Region:    "us-east-1",
			},
		},
		{
			name: "empty profile",
			session: Session{
				Provider:  "aws",
				Profile:   "",
				AccountID: "999999999999",
				Region:    "eu-west-1",
			},
		},
		{
			name:    "missing file returns error",
			wantErr: true,
		},
		{
			name: "corrupt JSON returns error",
			setup: func(t *testing.T, home string) {
				dir := filepath.Join(home, ".haven")
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "session.json"), []byte("{invalid"), 0600); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: true,
		},
		{
			name: "overwrite returns latest session",
			setup: func(t *testing.T, home string) {
				old := Session{
					Provider:  "aws",
					Profile:   "old-profile",
					AccountID: "111111111111",
					Region:    "us-west-2",
				}
				if err := SaveSession(old); err != nil {
					t.Fatal(err)
				}
			},
			session: Session{
				Provider:  "aws",
				Profile:   "new-profile",
				AccountID: "222222222222",
				Region:    "ap-southeast-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)

			if tt.setup != nil {
				tt.setup(t, tmp)
			}

			if tt.wantErr {
				_, err := LoadSession()
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err := SaveSession(tt.session); err != nil {
				t.Fatalf("SaveSession() error: %v", err)
			}

			got, err := LoadSession()
			if err != nil {
				t.Fatalf("LoadSession() error: %v", err)
			}

			if got.Provider != tt.session.Provider {
				t.Errorf("Provider = %q, want %q", got.Provider, tt.session.Provider)
			}
			if got.Profile != tt.session.Profile {
				t.Errorf("Profile = %q, want %q", got.Profile, tt.session.Profile)
			}
			if got.AccountID != tt.session.AccountID {
				t.Errorf("AccountID = %q, want %q", got.AccountID, tt.session.AccountID)
			}
			if got.Region != tt.session.Region {
				t.Errorf("Region = %q, want %q", got.Region, tt.session.Region)
			}
		})
	}
}
