package models

import (
	"strings"
	"testing"
)

func TestLookup_Known(t *testing.T) {
	cases := []struct {
		name         string
		wantRuntime  Runtime
		wantTag      string
		wantInstance string
		wantRAM      int
	}{
		{"llama3.2:1b", RuntimeOllama, "llama3.2:1b", "t3.large", 8},
		{"llama3.2:3b", RuntimeOllama, "llama3.2:3b", "t3.xlarge", 16},
		{"phi3:mini", RuntimeOllama, "phi3:mini", "t3.large", 8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Lookup(tc.name)
			if err != nil {
				t.Fatalf("Lookup(%q) returned error: %v", tc.name, err)
			}
			if cfg.Runtime != tc.wantRuntime {
				t.Errorf("Runtime = %q, want %q", cfg.Runtime, tc.wantRuntime)
			}
			if cfg.Tag != tc.wantTag {
				t.Errorf("Tag = %q, want %q", cfg.Tag, tc.wantTag)
			}
			if cfg.InstanceType != tc.wantInstance {
				t.Errorf("InstanceType = %q, want %q", cfg.InstanceType, tc.wantInstance)
			}
			if cfg.MinRAMGB != tc.wantRAM {
				t.Errorf("MinRAMGB = %d, want %d", cfg.MinRAMGB, tc.wantRAM)
			}
		})
	}
}

func TestLookup_Unknown(t *testing.T) {
	_, err := Lookup("nonexistent:model")
	if err == nil {
		t.Fatal("expected error for unknown model, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "nonexistent:model") {
		t.Errorf("error %q should contain model name", msg)
	}
	for _, known := range []string{"llama3.2:1b", "llama3.2:3b", "phi3:mini"} {
		if !strings.Contains(msg, known) {
			t.Errorf("error %q should list available model %q", msg, known)
		}
	}
}

func TestList(t *testing.T) {
	configs := List()
	if len(configs) != 3 {
		t.Errorf("List() returned %d configs, want 3", len(configs))
	}
}
