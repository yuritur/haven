package aws

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/mock"
)

func TestUpsertINISection(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		section string
		content string
		want    string
	}{
		{
			name:    "create new file",
			section: "default",
			content: "key = value\n",
			want:    "[default]\nkey = value\n",
		},
		{
			name:    "append to existing file",
			initial: "[default]\nfoo = bar\n",
			section: "other",
			content: "key = value\n",
			want:    "[default]\nfoo = bar\n\n[other]\nkey = value\n",
		},
		{
			name:    "replace existing section",
			initial: "[default]\nold = stuff\n",
			section: "default",
			content: "new = things\n",
			want:    "[default]\nnew = things\n",
		},
		{
			name:    "replace section preserves others",
			initial: "[first]\na = 1\n\n[second]\nb = 2\n\n[third]\nc = 3\n",
			section: "second",
			content: "b = 99\n",
			want:    "[first]\na = 1\n\n[second]\nb = 99\n[third]\nc = 3\n",
		},
		{
			name:    "profile haven section",
			initial: "[default]\nregion = us-west-2\n",
			section: "profile haven",
			content: "region = us-east-1\n",
			want:    "[default]\nregion = us-west-2\n\n[profile haven]\nregion = us-east-1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.ini")

			if tt.initial != "" {
				if err := os.WriteFile(path, []byte(tt.initial), 0600); err != nil {
					t.Fatal(err)
				}
			}

			if err := upsertINISection(path, tt.section, tt.content); err != nil {
				t.Fatalf("upsertINISection() error: %v", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", string(got), tt.want)
			}
		})
	}
}

func TestParseINISections(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name: "missing file returns nil",
		},
		{
			name:    "single section",
			content: "[default]\nkey = val\n",
			want:    []string{"default"},
		},
		{
			name:    "multiple sections",
			content: "[default]\na = 1\n\n[profile haven]\nb = 2\n\n[other]\nc = 3\n",
			want:    []string{"default", "profile haven", "other"},
		},
		{
			name:    "ignores non-section lines",
			content: "# comment\n[section]\nkey = val\nnot a section\n",
			want:    []string{"section"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config")

			if tt.content != "" {
				if err := os.WriteFile(path, []byte(tt.content), 0600); err != nil {
					t.Fatal(err)
				}
			}

			got := parseINISections(path)

			if tt.want == nil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("section[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCollectCredentials(t *testing.T) {
	tests := []struct {
		name    string
		input   func(string) string
		secret  func(string) string
		wantKey string
		wantSec string
		wantReg string
		wantErr bool
	}{
		{
			name: "valid with explicit region",
			input: func(p string) string {
				if p == "AWS Access Key ID" {
					return "AKID1234"
				}
				if p == "Region [us-east-1]" {
					return "eu-west-1"
				}
				return ""
			},
			secret:  func(string) string { return "SECRET" },
			wantKey: "AKID1234",
			wantSec: "SECRET",
			wantReg: "eu-west-1",
		},
		{
			name: "default region when empty",
			input: func(p string) string {
				if p == "AWS Access Key ID" {
					return "AKID"
				}
				return ""
			},
			secret:  func(string) string { return "SEC" },
			wantKey: "AKID",
			wantSec: "SEC",
			wantReg: "us-east-1",
		},
		{
			name:    "empty access key returns error",
			input:   func(string) string { return "" },
			secret:  func(string) string { return "SEC" },
			wantErr: true,
		},
		{
			name: "empty secret key returns error",
			input: func(p string) string {
				if p == "AWS Access Key ID" {
					return "AKID"
				}
				return ""
			},
			secret:  func(string) string { return "" },
			wantErr: true,
		},
		{
			name:    "whitespace-only access key returns error",
			input:   func(string) string { return "   " },
			secret:  func(string) string { return "SEC" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &mock.Prompter{
				InputFn:  tt.input,
				SecretFn: tt.secret,
				PrintFn:  func(string) {},
			}

			key, sec, reg, err := collectCredentials(p)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if key != tt.wantKey {
				t.Errorf("accessKey = %q, want %q", key, tt.wantKey)
			}
			if sec != tt.wantSec {
				t.Errorf("secretKey = %q, want %q", sec, tt.wantSec)
			}
			if reg != tt.wantReg {
				t.Errorf("region = %q, want %q", reg, tt.wantReg)
			}
		})
	}
}

func TestConfirmIdentity(t *testing.T) {
	tests := []struct {
		name    string
		confirm bool
		want    bool
	}{
		{name: "user confirms", confirm: true, want: true},
		{name: "user declines", confirm: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var printed []string
			p := &mock.Prompter{
				PrintFn:   func(s string) { printed = append(printed, s) },
				ConfirmFn: func(string) bool { return tt.confirm },
			}

			id := provider.Identity{
				AccountID: "123456789",
				Region:    "us-east-1",
				ARN:       "arn:aws:iam::123456789:user/test",
			}

			got := confirmIdentity(p, id)
			if got != tt.want {
				t.Errorf("confirmIdentity() = %v, want %v", got, tt.want)
			}

			all := strings.Join(printed, "\n")
			if !strings.Contains(all, id.AccountID) {
				t.Errorf("printed output missing account ID %q", id.AccountID)
			}
			if !strings.Contains(all, id.Region) {
				t.Errorf("printed output missing region %q", id.Region)
			}
			if !strings.Contains(all, id.ARN) {
				t.Errorf("printed output missing ARN %q", id.ARN)
			}
		})
	}
}
