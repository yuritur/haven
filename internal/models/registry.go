package models

import (
	"fmt"
	"sort"
	"strings"
)

type Runtime string

const (
	RuntimeOllama   Runtime = "ollama"
	RuntimeLlamaCpp Runtime = "llamacpp"
)

type Config struct {
	Runtime      Runtime
	Tag          string // runtime-specific model identifier (Ollama)
	InstanceType string
	MinRAMGB     int
	EBSVolumeGB  int
	HFRepo       string // HuggingFace repo for llamacpp (e.g. "bartowski/Llama-3.2-1B-Instruct-GGUF")
	HFFile       string // GGUF filename within the repo
}

func IsGPUInstance(instanceType string) bool {
	for _, prefix := range []string{"g4dn.", "g5.", "g5g.", "g6.", "p3.", "p4.", "p5."} {
		if strings.HasPrefix(instanceType, prefix) {
			return true
		}
	}
	return false
}

var registry = map[string]Config{
	"llama3.2:1b": {
		Runtime:      RuntimeOllama,
		Tag:          "llama3.2:1b",
		InstanceType: "t3.large",
		MinRAMGB:     8,
		EBSVolumeGB:  30,
		HFRepo:       "bartowski/Llama-3.2-1B-Instruct-GGUF",
		HFFile:       "Llama-3.2-1B-Instruct-Q4_K_M.gguf",
	},
	"llama3.2:3b": {
		Runtime:      RuntimeOllama,
		Tag:          "llama3.2:3b",
		InstanceType: "t3.xlarge",
		MinRAMGB:     16,
		EBSVolumeGB:  30,
		HFRepo:       "bartowski/Llama-3.2-3B-Instruct-GGUF",
		HFFile:       "Llama-3.2-3B-Instruct-Q4_K_M.gguf",
	},
	"phi3:mini": {
		Runtime:      RuntimeOllama,
		Tag:          "phi3:mini",
		InstanceType: "t3.large",
		MinRAMGB:     8,
		EBSVolumeGB:  30,
		HFRepo:       "bartowski/Phi-3-mini-4k-instruct-GGUF",
		HFFile:       "Phi-3-mini-4k-instruct-Q4_K_M.gguf",
	},
	"qwen3.5:4b": {
		Runtime:      RuntimeOllama,
		Tag:          "qwen3.5:4b",
		InstanceType: "g5.xlarge",
		MinRAMGB:     16,
		EBSVolumeGB:  80,
		HFRepo:       "Qwen/Qwen2.5-3B-Instruct-GGUF",
		HFFile:       "qwen2.5-3b-instruct-q4_k_m.gguf",
	},
	"qwen3.5:9b": {
		Runtime:      RuntimeOllama,
		Tag:          "qwen3.5:9b",
		InstanceType: "g5.xlarge",
		MinRAMGB:     16,
		EBSVolumeGB:  100,
		HFRepo:       "Qwen/Qwen2.5-7B-Instruct-GGUF",
		HFFile:       "qwen2.5-7b-instruct-q4_k_m.gguf",
	},
	"qwen3.5:27b": {
		Runtime:      RuntimeOllama,
		Tag:          "qwen3.5:27b",
		InstanceType: "g5.2xlarge",
		MinRAMGB:     32,
		EBSVolumeGB:  100,
		HFRepo:       "Qwen/Qwen2.5-32B-Instruct-GGUF",
		HFFile:       "qwen2.5-32b-instruct-q4_k_m.gguf",
	},
}

func Lookup(name string) (Config, error) {
	cfg, ok := registry[name]
	if !ok {
		names := make([]string, 0, len(registry))
		for k := range registry {
			names = append(names, k)
		}
		sort.Strings(names)
		return Config{}, fmt.Errorf("unknown model %q — available: %s", name, strings.Join(names, ", "))
	}
	return cfg, nil
}

func LookupWithRuntime(name string, runtimeOverride Runtime) (Config, error) {
	cfg, err := Lookup(name)
	if err != nil {
		return Config{}, err
	}
	cfg.Runtime = runtimeOverride
	if runtimeOverride == RuntimeLlamaCpp {
		if cfg.HFRepo == "" || cfg.HFFile == "" {
			return Config{}, fmt.Errorf("model %q has no HuggingFace GGUF mapping for llamacpp runtime", name)
		}
	}
	return cfg, nil
}

func List() []Config {
	result := make([]Config, 0, len(registry))
	for _, cfg := range registry {
		result = append(result, cfg)
	}
	return result
}
