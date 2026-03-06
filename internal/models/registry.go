package models

import "fmt"

type Config struct {
	Name         string
	OllamaTag    string
	InstanceType string
	MinRAMGB     int
}

var registry = map[string]Config{
	"llama3.2:1b": {
		Name:         "llama3.2:1b",
		OllamaTag:    "llama3.2:1b",
		InstanceType: "t3.large",
		MinRAMGB:     8,
	},
	"llama3.2:3b": {
		Name:         "llama3.2:3b",
		OllamaTag:    "llama3.2:3b",
		InstanceType: "t3.xlarge",
		MinRAMGB:     16,
	},
	"phi3:mini": {
		Name:         "phi3:mini",
		OllamaTag:    "phi3:mini",
		InstanceType: "t3.large",
		MinRAMGB:     8,
	},
}

func Lookup(name string) (Config, error) {
	cfg, ok := registry[name]
	if !ok {
		return Config{}, fmt.Errorf("unknown model %q — available: llama3.2:1b, llama3.2:3b, phi3:mini", name)
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
