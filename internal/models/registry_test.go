package models

import (
	"strings"
	"testing"
)

func TestLookup_Known(t *testing.T) {
	cases := []struct {
		name       string
		wantTag    string
		wantHFRepo string
		wantHFFile string
	}{
		{"llama3.2:1b", "llama3.2:1b", "bartowski/Llama-3.2-1B-Instruct-GGUF", "Llama-3.2-1B-Instruct-Q4_K_M.gguf"},
		{"llama3.2:3b", "llama3.2:3b", "bartowski/Llama-3.2-3B-Instruct-GGUF", "Llama-3.2-3B-Instruct-Q4_K_M.gguf"},
		{"phi3:mini", "phi3:mini", "bartowski/Phi-3-mini-4k-instruct-GGUF", "Phi-3-mini-4k-instruct-Q4_K_M.gguf"},
		{"qwen3.5:4b", "qwen3.5:4b", "Qwen/Qwen2.5-3B-Instruct-GGUF", "qwen2.5-3b-instruct-q4_k_m.gguf"},
		{"qwen3.5:9b", "qwen3.5:9b", "Qwen/Qwen2.5-7B-Instruct-GGUF", "qwen2.5-7b-instruct-q4_k_m.gguf"},
		{"qwen3.5:27b", "qwen3.5:27b", "Qwen/Qwen2.5-32B-Instruct-GGUF", "qwen2.5-32b-instruct-q4_k_m.gguf"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Lookup(tc.name)
			if err != nil {
				t.Fatalf("Lookup(%q) returned error: %v", tc.name, err)
			}
			if cfg.Ollama == nil {
				t.Fatal("expected non-nil Ollama config")
			}
			if cfg.Ollama.Tag != tc.wantTag {
				t.Errorf("Ollama.Tag = %q, want %q", cfg.Ollama.Tag, tc.wantTag)
			}
			if cfg.LlamaCpp == nil {
				t.Fatal("expected non-nil LlamaCpp config")
			}
			if cfg.LlamaCpp.HFRepo != tc.wantHFRepo {
				t.Errorf("LlamaCpp.HFRepo = %q, want %q", cfg.LlamaCpp.HFRepo, tc.wantHFRepo)
			}
			if cfg.LlamaCpp.HFFile != tc.wantHFFile {
				t.Errorf("LlamaCpp.HFFile = %q, want %q", cfg.LlamaCpp.HFFile, tc.wantHFFile)
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

func TestSupportsRuntime(t *testing.T) {
	cases := []struct {
		name    string
		config  Config
		runtime RuntimeName
		want    bool
	}{
		{
			"ollama supported",
			Config{Ollama: &OllamaConfig{Tag: "test"}},
			Ollama,
			true,
		},
		{
			"llamacpp supported",
			Config{LlamaCpp: &LlamaCppConfig{HFRepo: "r", HFFile: "f"}},
			LlamaCpp,
			true,
		},
		{
			"ollama not supported",
			Config{LlamaCpp: &LlamaCppConfig{HFRepo: "r", HFFile: "f"}},
			Ollama,
			false,
		},
		{
			"llamacpp not supported",
			Config{Ollama: &OllamaConfig{Tag: "test"}},
			LlamaCpp,
			false,
		},
		{
			"both supported ollama",
			Config{Ollama: &OllamaConfig{Tag: "test"}, LlamaCpp: &LlamaCppConfig{HFRepo: "r", HFFile: "f"}},
			Ollama,
			true,
		},
		{
			"unknown runtime",
			Config{Ollama: &OllamaConfig{Tag: "test"}},
			RuntimeName("vllm"),
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.config.SupportsRuntime(tc.runtime)
			if got != tc.want {
				t.Errorf("SupportsRuntime(%q) = %v, want %v", tc.runtime, got, tc.want)
			}
		})
	}
}
