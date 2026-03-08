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
		wantEBS      int
	}{
		{"llama3.2:1b", RuntimeOllama, "llama3.2:1b", "t3.large", 8, 30},
		{"llama3.2:3b", RuntimeOllama, "llama3.2:3b", "t3.xlarge", 16, 30},
		{"phi3:mini", RuntimeOllama, "phi3:mini", "t3.large", 8, 30},
		{"qwen3.5:4b", RuntimeOllama, "qwen3.5:4b", "g5.xlarge", 16, 60},
		{"qwen3.5:9b", RuntimeOllama, "qwen3.5:9b", "g5.xlarge", 16, 80},
		{"qwen3.5:27b", RuntimeOllama, "qwen3.5:27b", "g5.2xlarge", 32, 100},
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
			if cfg.EBSVolumeGB != tc.wantEBS {
				t.Errorf("EBSVolumeGB = %d, want %d", cfg.EBSVolumeGB, tc.wantEBS)
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
	if len(configs) != 6 {
		t.Errorf("List() returned %d configs, want 6", len(configs))
	}
}

func TestIsGPUInstance(t *testing.T) {
	cases := []struct {
		instanceType string
		want         bool
	}{
		{"g5.xlarge", true},
		{"g5.2xlarge", true},
		{"g4dn.xlarge", true},
		{"p3.2xlarge", true},
		{"t3.large", false},
		{"t3.xlarge", false},
		{"m5.large", false},
	}
	for _, tc := range cases {
		t.Run(tc.instanceType, func(t *testing.T) {
			got := IsGPUInstance(tc.instanceType)
			if got != tc.want {
				t.Errorf("IsGPUInstance(%q) = %v, want %v", tc.instanceType, got, tc.want)
			}
		})
	}
}
