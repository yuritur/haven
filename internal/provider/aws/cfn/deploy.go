package cfn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go"

	"github.com/havenapp/haven/internal/models"
)

type DeployInput struct {
	StackName    string
	Runtime      models.RuntimeName
	ModelTag     string
	InstanceType string
	UserIP       string
	APIKey       string
	TLSCert      string
	TLSKey       string
	EBSVolumeGB  int
	HFRepo       string
	HFFile       string
	GPU          bool
	Out          io.Writer
}

type DeployResult struct {
	StackName  string
	InstanceID string
	PublicIP   string
}

func Deploy(ctx context.Context, cfg aws.Config, input DeployInput) (DeployResult, error) {
	templateJSON, err := GenerateTemplate(TemplateInput{
		UserIP:       input.UserIP,
		APIKey:       input.APIKey,
		Runtime:      input.Runtime,
		ModelTag:     input.ModelTag,
		InstanceType: input.InstanceType,
		TLSCert:      input.TLSCert,
		TLSKey:       input.TLSKey,
		EBSVolumeGB:  input.EBSVolumeGB,
		HFRepo:       input.HFRepo,
		HFFile:       input.HFFile,
		GPU:          input.GPU,
	})
	if err != nil {
		return DeployResult{}, fmt.Errorf("generate template: %w", err)
	}

	cfnClient := cloudformation.NewFromConfig(cfg)

	_, err = cfnClient.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:    aws.String(input.StackName),
		TemplateBody: aws.String(templateJSON),
		Capabilities: []cfntypes.Capability{cfntypes.CapabilityCapabilityIam},
	})
	if err != nil {
		return DeployResult{}, fmt.Errorf("create stack %s: %w", input.StackName, err)
	}

	if err := pollStackEvents(ctx, cfnClient, input.StackName, input.Out, isDeployTerminal); err != nil {
		return DeployResult{}, err
	}

	out, err := cfnClient.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(input.StackName),
	})
	if err != nil {
		return DeployResult{}, fmt.Errorf("describe stack %s: %w", input.StackName, err)
	}
	if len(out.Stacks) == 0 {
		return DeployResult{}, fmt.Errorf("stack %s not found after creation", input.StackName)
	}

	stack := out.Stacks[0]
	if stack.StackStatus != cfntypes.StackStatusCreateComplete {
		return DeployResult{}, fmt.Errorf("stack %s ended in status %s", input.StackName, stack.StackStatus)
	}

	result := DeployResult{StackName: input.StackName}
	for _, o := range stack.Outputs {
		switch aws.ToString(o.OutputKey) {
		case "InstanceId":
			result.InstanceID = aws.ToString(o.OutputValue)
		case "PublicIP":
			result.PublicIP = aws.ToString(o.OutputValue)
		}
	}

	return result, nil
}

func isDeployTerminal(status cfntypes.StackStatus) (done bool, failed bool) {
	switch status {
	case cfntypes.StackStatusCreateComplete:
		return true, false
	case cfntypes.StackStatusCreateFailed,
		cfntypes.StackStatusRollbackComplete,
		cfntypes.StackStatusRollbackFailed:
		return true, true
	}
	return false, false
}

func pollStackEvents(
	ctx context.Context,
	cfnClient *cloudformation.Client,
	stackName string,
	out io.Writer,
	isTerminal func(cfntypes.StackStatus) (done bool, failed bool),
) error {
	seenEventIDs := map[string]bool{}

	for {
		eventsOut, err := cfnClient.DescribeStackEvents(ctx, &cloudformation.DescribeStackEventsInput{
			StackName: aws.String(stackName),
		})
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ValidationError" && strings.Contains(apiErr.ErrorMessage(), "does not exist") {
				done, failed := isTerminal(cfntypes.StackStatusDeleteComplete)
				if done && !failed {
					return nil
				}
			}
			return fmt.Errorf("describe stack events for %s: %w", stackName, err)
		}

		var newEvents []cfntypes.StackEvent
		for _, e := range eventsOut.StackEvents {
			if !seenEventIDs[aws.ToString(e.EventId)] {
				newEvents = append(newEvents, e)
			}
		}
		for i := len(newEvents) - 1; i >= 0; i-- {
			e := newEvents[i]
			seenEventIDs[aws.ToString(e.EventId)] = true
			ts := ""
			if e.Timestamp != nil {
				ts = e.Timestamp.Format(time.RFC3339)
			}
			fmt.Fprintf(out, "  [%s] %s %s\n",
				ts,
				aws.ToString(e.ResourceType),
				string(e.ResourceStatus),
			)
		}

		stackOut, err := cfnClient.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
			StackName: aws.String(stackName),
		})
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ValidationError" && strings.Contains(apiErr.ErrorMessage(), "does not exist") {
				done, failed := isTerminal(cfntypes.StackStatusDeleteComplete)
				if done && !failed {
					return nil
				}
			}
			return fmt.Errorf("describe stack %s: %w", stackName, err)
		}
		if len(stackOut.Stacks) == 0 {
			done, failed := isTerminal(cfntypes.StackStatusDeleteComplete)
			if done && !failed {
				return nil
			}
			return fmt.Errorf("stack %s disappeared during polling", stackName)
		}

		done, failed := isTerminal(stackOut.Stacks[0].StackStatus)
		if done {
			if failed {
				return fmt.Errorf("stack %s reached terminal failure status: %s", stackName, stackOut.Stacks[0].StackStatus)
			}
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}
