package aws

import (
	"testing"

	"github.com/havenapp/haven/internal/models"
)

func TestResolveInstance_Known(t *testing.T) {
	cases := []struct {
		model        string
		runtime      models.Runtime
		wantInstance string
		wantEBS      int
		wantGPU      bool
	}{
		{"llama3.2:1b", models.RuntimeOllama, "t3.large", 30, false},
		{"llama3.2:1b", models.RuntimeLlamaCpp, "t3.large", 30, false},
		{"llama3.2:3b", models.RuntimeOllama, "t3.xlarge", 30, false},
		{"llama3.2:3b", models.RuntimeLlamaCpp, "t3.xlarge", 30, false},
		{"phi3:mini", models.RuntimeOllama, "t3.large", 30, false},
		{"phi3:mini", models.RuntimeLlamaCpp, "t3.large", 30, false},
		{"qwen3.5:4b", models.RuntimeOllama, "g5.xlarge", 80, true},
		{"qwen3.5:4b", models.RuntimeLlamaCpp, "g5.xlarge", 80, true},
		{"qwen3.5:9b", models.RuntimeOllama, "g5.xlarge", 100, true},
		{"qwen3.5:9b", models.RuntimeLlamaCpp, "g5.xlarge", 100, true},
		{"qwen3.5:27b", models.RuntimeOllama, "g5.2xlarge", 100, true},
		{"qwen3.5:27b", models.RuntimeLlamaCpp, "g5.2xlarge", 100, true},
	}
	for _, tc := range cases {
		t.Run(tc.model+"_"+string(tc.runtime), func(t *testing.T) {
			spec, err := ResolveInstance(tc.model, tc.runtime)
			if err != nil {
				t.Fatalf("ResolveInstance(%q, %q) returned error: %v", tc.model, tc.runtime, err)
			}
			if spec.InstanceType != tc.wantInstance {
				t.Errorf("InstanceType = %q, want %q", spec.InstanceType, tc.wantInstance)
			}
			if spec.EBSVolumeGB != tc.wantEBS {
				t.Errorf("EBSVolumeGB = %d, want %d", spec.EBSVolumeGB, tc.wantEBS)
			}
			if spec.GPU != tc.wantGPU {
				t.Errorf("GPU = %v, want %v", spec.GPU, tc.wantGPU)
			}
		})
	}
}

func TestResolveInstance_AllModelsResolvable(t *testing.T) {
	for _, name := range models.Names() {
		for _, rt := range []models.Runtime{models.RuntimeOllama, models.RuntimeLlamaCpp} {
			t.Run(name+"_"+string(rt), func(t *testing.T) {
				_, err := ResolveInstance(name, rt)
				if err != nil {
					t.Errorf("model %q registered in models.Names() but ResolveInstance fails: %v", name, err)
				}
			})
		}
	}
}

func TestResolveInstance_Unknown(t *testing.T) {
	_, err := ResolveInstance("nonexistent:model", models.RuntimeOllama)
	if err == nil {
		t.Fatal("expected error for unknown model, got nil")
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
			got := isGPUInstance(tc.instanceType)
			if got != tc.want {
				t.Errorf("isGPUInstance(%q) = %v, want %v", tc.instanceType, got, tc.want)
			}
		})
	}
}
