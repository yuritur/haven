package cfn

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

func Destroy(ctx context.Context, cfg aws.Config, stackName string, out io.Writer) error {
	cfnClient := cloudformation.NewFromConfig(cfg)

	_, err := cfnClient.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return fmt.Errorf("delete stack %s: %w", stackName, err)
	}

	return pollStackEvents(ctx, cfnClient, stackName, out, isDestroyTerminal)
}

func isDestroyTerminal(status cfntypes.StackStatus) (done bool, failed bool) {
	switch status {
	case cfntypes.StackStatusDeleteComplete:
		return true, false
	case cfntypes.StackStatusDeleteFailed:
		return true, true
	}
	return false, false
}
