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

type OllamaConfig struct {
	Tag string
}

type LlamaCppConfig struct {
	HFRepo string
	HFFile string
}

type Config struct {
	Ollama   *OllamaConfig
	LlamaCpp *LlamaCppConfig
}

func (c Config) SupportsRuntime(rt Runtime) bool {
	switch rt {
	case RuntimeOllama:
		return c.Ollama != nil
	case RuntimeLlamaCpp:
		return c.LlamaCpp != nil
	}
	return false
}

var registry = map[string]Config{
	"llama3.2:1b": {
		Ollama: &OllamaConfig{Tag: "llama3.2:1b"},
		LlamaCpp: &LlamaCppConfig{
			HFRepo: "bartowski/Llama-3.2-1B-Instruct-GGUF",
			HFFile: "Llama-3.2-1B-Instruct-Q4_K_M.gguf",
		},
	},
	"llama3.2:3b": {
		Ollama: &OllamaConfig{Tag: "llama3.2:3b"},
		LlamaCpp: &LlamaCppConfig{
			HFRepo: "bartowski/Llama-3.2-3B-Instruct-GGUF",
			HFFile: "Llama-3.2-3B-Instruct-Q4_K_M.gguf",
		},
	},
	"phi3:mini": {
		Ollama: &OllamaConfig{Tag: "phi3:mini"},
		LlamaCpp: &LlamaCppConfig{
			HFRepo: "bartowski/Phi-3-mini-4k-instruct-GGUF",
			HFFile: "Phi-3-mini-4k-instruct-Q4_K_M.gguf",
		},
	},
	"qwen3.5:4b": {
		Ollama: &OllamaConfig{Tag: "qwen3.5:4b"},
		LlamaCpp: &LlamaCppConfig{
			HFRepo: "Qwen/Qwen2.5-3B-Instruct-GGUF",
			HFFile: "qwen2.5-3b-instruct-q4_k_m.gguf",
		},
	},
	"qwen3.5:9b": {
		Ollama: &OllamaConfig{Tag: "qwen3.5:9b"},
		LlamaCpp: &LlamaCppConfig{
			HFRepo: "Qwen/Qwen2.5-7B-Instruct-GGUF",
			HFFile: "qwen2.5-7b-instruct-q4_k_m.gguf",
		},
	},
	"qwen3.5:27b": {
		Ollama: &OllamaConfig{Tag: "qwen3.5:27b"},
		LlamaCpp: &LlamaCppConfig{
			HFRepo: "Qwen/Qwen2.5-32B-Instruct-GGUF",
			HFFile: "qwen2.5-32b-instruct-q4_k_m.gguf",
		},
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

func List() []Config {
	result := make([]Config, 0, len(registry))
	for _, cfg := range registry {
		result = append(result, cfg)
	}
	return result
}

func Names() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
