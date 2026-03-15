package cli

import (
	"context"
	"fmt"

	"github.com/havenapp/haven/internal/provider"
)

func resolveDeployment(ctx context.Context, prov provider.Provider, prompter provider.Prompter, id string) (*provider.Deployment, error) {
	if id != "" {
		d, err := prov.LoadDeployment(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("load deployment: %w", err)
		}
		return d, nil
	}

	deployments, err := prov.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	if len(deployments) == 0 {
		return nil, fmt.Errorf("no active deployments — run 'haven deploy <model>' first")
	}
	if len(deployments) == 1 {
		return &deployments[0], nil
	}

	options := make([]string, len(deployments))
	for i, d := range deployments {
		options[i] = fmt.Sprintf("%s (%s [%s])", d.ID, d.Model, d.Runtime)
	}
	idx := prompter.Select("Select a deployment:", options)
	if idx < 0 {
		return nil, fmt.Errorf("no deployment selected")
	}
	return &deployments[idx], nil
}
