package aws

import (
	"fmt"
	"strings"

	"github.com/havenapp/haven/internal/models"
)

type InstanceSpec struct {
	InstanceType string
	EBSVolumeGB  int
	GPU          bool
}

func isGPUInstance(instanceType string) bool {
	for _, prefix := range []string{"g4dn.", "g5.", "g5g.", "g6.", "p3.", "p4.", "p5."} {
		if strings.HasPrefix(instanceType, prefix) {
			return true
		}
	}
	return false
}

var instanceTable = map[string]struct {
	InstanceType string
	EBSVolumeGB  int
}{
	"llama3.2:1b": {InstanceType: "t3.large", EBSVolumeGB: 30},
	"llama3.2:3b": {InstanceType: "t3.xlarge", EBSVolumeGB: 30},
	"phi3:mini":   {InstanceType: "t3.large", EBSVolumeGB: 30},
	"qwen3.5:4b":  {InstanceType: "g5.xlarge", EBSVolumeGB: 80},
	"qwen3.5:9b":  {InstanceType: "g5.xlarge", EBSVolumeGB: 100},
	"qwen3.5:27b": {InstanceType: "g5.2xlarge", EBSVolumeGB: 100},
}

func ResolveInstance(model string, _ models.Runtime) (InstanceSpec, error) {
	entry, ok := instanceTable[model]
	if !ok {
		return InstanceSpec{}, fmt.Errorf("no instance mapping for model %q", model)
	}
	return InstanceSpec{
		InstanceType: entry.InstanceType,
		EBSVolumeGB:  entry.EBSVolumeGB,
		GPU:          isGPUInstance(entry.InstanceType),
	}, nil
}
