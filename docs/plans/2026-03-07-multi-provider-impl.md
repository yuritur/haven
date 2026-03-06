# Multi-Provider Abstraction Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Decouple Haven from AWS by introducing `Provider` and `StateStore` interfaces, moving AWS implementation to `internal/provider/aws/`, and updating all CLI text to be provider-agnostic.

**Architecture:** `internal/provider/provider.go` defines interfaces and shared types. AWS implementation lives entirely under `internal/provider/aws/`. The CLI accepts `--provider aws` (default) and uses a factory function to instantiate the right provider.

**Tech Stack:** Go, cobra, aws-sdk-go-v2. No new dependencies.

---

### Task 1: Create `internal/provider/provider.go`

**Files:**
- Create: `internal/provider/provider.go`

**Step 1: Create the file**

```go
package provider

import (
	"context"
	"time"
)

type Identity struct {
	AccountID string
	Region    string
}

type DeployInput struct {
	DeploymentID string
	Model        string
	InstanceType string
	UserIP       string
	APIKey       string
}

type DeployResult struct {
	ProviderRef string
	InstanceID  string
	PublicIP    string
}

type Deployment struct {
	ID           string    `json:"deployment_id"`
	Provider     string    `json:"provider"`
	ProviderRef  string    `json:"provider_ref"`
	CreatedAt    time.Time `json:"created_at"`
	Region       string    `json:"region"`
	Model        string    `json:"model"`
	InstanceType string    `json:"instance_type"`
	InstanceID   string    `json:"instance_id"`
	PublicIP     string    `json:"public_ip"`
	Endpoint     string    `json:"endpoint"`
	APIKey       string    `json:"api_key"`
}

type Provider interface {
	Identity(ctx context.Context) (Identity, error)
	Deploy(ctx context.Context, input DeployInput) (DeployResult, error)
	Destroy(ctx context.Context, providerRef string) error
}

type StateStore interface {
	Save(ctx context.Context, d Deployment) error
	Load(ctx context.Context, id string) (*Deployment, error)
	List(ctx context.Context) ([]Deployment, error)
	Delete(ctx context.Context, id string) error
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/provider/...
```
Expected: no output (success).

---

### Task 2: Create `internal/provider/aws/credentials.go`

**Files:**
- Create: `internal/provider/aws/credentials.go`

Content is identical to `internal/aws/credentials.go` — same package name `aws`, same imports, same logic:

```go
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type identity struct {
	AccountID string
	Region    string
}

func loadConfig(ctx context.Context) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load AWS config: %w", err)
	}
	return cfg, nil
}

func getIdentity(ctx context.Context, cfg aws.Config) (identity, error) {
	client := sts.NewFromConfig(cfg)
	out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return identity{}, fmt.Errorf("GetCallerIdentity failed — check AWS credentials: %w", err)
	}
	return identity{
		AccountID: aws.ToString(out.Account),
		Region:    cfg.Region,
	}, nil
}
```

Note: functions are lowercase (unexported) — they're internal implementation details, only used by `provider.go` in the same package.

**Step 2: Verify**

```bash
go build ./internal/provider/aws/...
```

---

### Task 3: Create `internal/provider/aws/bootstrap.go`

**Files:**
- Create: `internal/provider/aws/bootstrap.go`

Content identical to `internal/aws/bootstrap.go`, just package is `aws` (same as before):

```go
package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func stateBucketName(accountID string) string {
	return fmt.Sprintf("haven-state-%s", accountID)
}

// ensureStateBucket creates the S3 state bucket if it doesn't exist.
// Safe to call on every deploy — idempotent.
func ensureStateBucket(ctx context.Context, cfg aws.Config, accountID string) (string, error) {
	bucketName := stateBucketName(accountID)
	client := s3.NewFromConfig(cfg)

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	// us-east-1 must NOT have LocationConstraint; all other regions must.
	if cfg.Region != "us-east-1" {
		input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(cfg.Region),
		}
	}

	_, err := client.CreateBucket(ctx, input)
	if err != nil {
		var alreadyOwned *s3types.BucketAlreadyOwnedByYou
		var alreadyExists *s3types.BucketAlreadyExists
		if errors.As(err, &alreadyOwned) || errors.As(err, &alreadyExists) {
			return bucketName, nil
		}
		return "", fmt.Errorf("create state bucket %s: %w", bucketName, err)
	}

	_, err = client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucketName),
		VersioningConfiguration: &s3types.VersioningConfiguration{
			Status: s3types.BucketVersioningStatusEnabled,
		},
	})
	if err != nil {
		return "", fmt.Errorf("enable versioning on %s: %w", bucketName, err)
	}

	on := true
	_, err = client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
		PublicAccessBlockConfiguration: &s3types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       &on,
			BlockPublicPolicy:     &on,
			IgnorePublicAcls:      &on,
			RestrictPublicBuckets: &on,
		},
	})
	if err != nil {
		return "", fmt.Errorf("block public access on %s: %w", bucketName, err)
	}

	return bucketName, nil
}
```

Note: functions are lowercase — internal to this package.

**Step 2: Verify**

```bash
go build ./internal/provider/aws/...
```

---

### Task 4: Create `internal/provider/aws/cfn/template.go`

**Files:**
- Create: `internal/provider/aws/cfn/template.go`

Almost identical to `internal/cfn/template.go`. Two changes:
1. Add `InstanceType string` to `TemplateInput`
2. Replace hardcoded `"t3.large"` with `input.InstanceType`

```go
package cfn

import (
	"encoding/json"
	"strings"
)

type TemplateInput struct {
	UserIP       string
	APIKey       string
	Model        string
	InstanceType string
}

const userDataTemplate = `#!/bin/bash
set -e
exec > /var/log/haven-bootstrap.log 2>&1

echo "Installing Ollama..."
curl -fsSL https://ollama.com/install.sh | sh

echo "Configuring Ollama..."
mkdir -p /etc/systemd/system/ollama.service.d
cat > /etc/systemd/system/ollama.service.d/override.conf << 'CONF'
[Service]
Environment="OLLAMA_HOST=0.0.0.0:11434"
Environment="OLLAMA_API_KEY=HAVEN_API_KEY"
CONF

systemctl daemon-reload
systemctl enable ollama
systemctl start ollama

echo "Waiting for Ollama to start..."
for i in $(seq 1 30); do
    curl -sf http://localhost:11434/api/tags > /dev/null 2>&1 && break
    sleep 2
done

echo "Pulling model HAVEN_MODEL..."
ollama pull HAVEN_MODEL
echo "Bootstrap complete."
`

func GenerateTemplate(input TemplateInput) (string, error) {
	userData := strings.ReplaceAll(userDataTemplate, "HAVEN_API_KEY", input.APIKey)
	userData = strings.ReplaceAll(userData, "HAVEN_MODEL", input.Model)

	template := map[string]interface{}{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Description":              "Haven deployment",
		"Parameters": map[string]interface{}{
			"LatestAmiId": map[string]interface{}{
				"Type":    "AWS::SSM::Parameter::Value<AWS::EC2::Image::Id>",
				"Default": "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64",
			},
		},
		"Resources": map[string]interface{}{
			"HavenVPC": map[string]interface{}{
				"Type": "AWS::EC2::VPC",
				"Properties": map[string]interface{}{
					"CidrBlock":          "10.0.0.0/16",
					"EnableDnsHostnames": true,
					"EnableDnsSupport":   true,
				},
			},
			"HavenSubnet": map[string]interface{}{
				"Type": "AWS::EC2::Subnet",
				"Properties": map[string]interface{}{
					"VpcId":               map[string]interface{}{"Ref": "HavenVPC"},
					"CidrBlock":           "10.0.1.0/24",
					"MapPublicIpOnLaunch": true,
				},
			},
			"HavenIGW": map[string]interface{}{
				"Type": "AWS::EC2::InternetGateway",
			},
			"HavenVPCGWAttachment": map[string]interface{}{
				"Type": "AWS::EC2::VPCGatewayAttachment",
				"Properties": map[string]interface{}{
					"VpcId":             map[string]interface{}{"Ref": "HavenVPC"},
					"InternetGatewayId": map[string]interface{}{"Ref": "HavenIGW"},
				},
			},
			"HavenRouteTable": map[string]interface{}{
				"Type": "AWS::EC2::RouteTable",
				"Properties": map[string]interface{}{
					"VpcId": map[string]interface{}{"Ref": "HavenVPC"},
				},
			},
			"HavenRoute": map[string]interface{}{
				"Type":      "AWS::EC2::Route",
				"DependsOn": "HavenVPCGWAttachment",
				"Properties": map[string]interface{}{
					"RouteTableId":         map[string]interface{}{"Ref": "HavenRouteTable"},
					"DestinationCidrBlock": "0.0.0.0/0",
					"GatewayId":            map[string]interface{}{"Ref": "HavenIGW"},
				},
			},
			"HavenSubnetRTAssoc": map[string]interface{}{
				"Type": "AWS::EC2::SubnetRouteTableAssociation",
				"Properties": map[string]interface{}{
					"SubnetId":     map[string]interface{}{"Ref": "HavenSubnet"},
					"RouteTableId": map[string]interface{}{"Ref": "HavenRouteTable"},
				},
			},
			"HavenSG": map[string]interface{}{
				"Type": "AWS::EC2::SecurityGroup",
				"Properties": map[string]interface{}{
					"GroupDescription": "Haven security group",
					"VpcId":            map[string]interface{}{"Ref": "HavenVPC"},
					"SecurityGroupIngress": []interface{}{
						map[string]interface{}{
							"IpProtocol": "tcp",
							"FromPort":   11434,
							"ToPort":     11434,
							"CidrIp":     input.UserIP,
						},
					},
					"SecurityGroupEgress": []interface{}{
						map[string]interface{}{
							"IpProtocol": "-1",
							"CidrIp":     "0.0.0.0/0",
						},
					},
				},
			},
			"HavenRole": map[string]interface{}{
				"Type": "AWS::IAM::Role",
				"Properties": map[string]interface{}{
					"AssumeRolePolicyDocument": map[string]interface{}{
						"Version": "2012-10-17",
						"Statement": []interface{}{
							map[string]interface{}{
								"Effect": "Allow",
								"Principal": map[string]interface{}{
									"Service": "ec2.amazonaws.com",
								},
								"Action": "sts:AssumeRole",
							},
						},
					},
					"ManagedPolicyArns": []interface{}{
						"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
					},
				},
			},
			"HavenInstanceProfile": map[string]interface{}{
				"Type": "AWS::IAM::InstanceProfile",
				"Properties": map[string]interface{}{
					"Roles": []interface{}{
						map[string]interface{}{"Ref": "HavenRole"},
					},
				},
			},
			"HavenInstance": map[string]interface{}{
				"Type": "AWS::EC2::Instance",
				"Properties": map[string]interface{}{
					"ImageId":            map[string]interface{}{"Ref": "LatestAmiId"},
					"InstanceType":       input.InstanceType,
					"SubnetId":           map[string]interface{}{"Ref": "HavenSubnet"},
					"SecurityGroupIds":   []interface{}{map[string]interface{}{"Ref": "HavenSG"}},
					"IamInstanceProfile": map[string]interface{}{"Ref": "HavenInstanceProfile"},
					"BlockDeviceMappings": []interface{}{
						map[string]interface{}{
							"DeviceName": "/dev/xvda",
							"Ebs": map[string]interface{}{
								"VolumeSize": 30,
								"VolumeType": "gp3",
								"Encrypted":  true,
							},
						},
					},
					"MetadataOptions": map[string]interface{}{
						"HttpTokens": "required",
					},
					"UserData": map[string]interface{}{
						"Fn::Base64": userData,
					},
				},
			},
			"HavenEIP": map[string]interface{}{
				"Type": "AWS::EC2::EIP",
				"Properties": map[string]interface{}{
					"Domain": "vpc",
				},
			},
			"HavenEIPAssoc": map[string]interface{}{
				"Type": "AWS::EC2::EIPAssociation",
				"Properties": map[string]interface{}{
					"InstanceId":   map[string]interface{}{"Ref": "HavenInstance"},
					"AllocationId": map[string]interface{}{"Fn::GetAtt": []interface{}{"HavenEIP", "AllocationId"}},
				},
			},
		},
		"Outputs": map[string]interface{}{
			"InstanceId": map[string]interface{}{
				"Value": map[string]interface{}{"Ref": "HavenInstance"},
			},
			"PublicIP": map[string]interface{}{
				"Value": map[string]interface{}{"Ref": "HavenEIP"},
			},
		},
	}

	buf, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
```

**Step 2: Verify**

```bash
go build ./internal/provider/aws/cfn/...
```

---

### Task 5: Create `internal/provider/aws/cfn/deploy.go`

**Files:**
- Create: `internal/provider/aws/cfn/deploy.go`

Same as `internal/cfn/deploy.go` but `DeployInput` gains `InstanceType` and passes it to `GenerateTemplate`:

```go
package cfn

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type DeployInput struct {
	StackName    string
	Model        string
	InstanceType string
	UserIP       string
	APIKey       string
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
		Model:        input.Model,
		InstanceType: input.InstanceType,
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

	if err := pollStackEvents(ctx, cfnClient, input.StackName, isDeployTerminal); err != nil {
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
	isTerminal func(cfntypes.StackStatus) (done bool, failed bool),
) error {
	seenEventIDs := map[string]bool{}

	for {
		eventsOut, err := cfnClient.DescribeStackEvents(ctx, &cloudformation.DescribeStackEventsInput{
			StackName: aws.String(stackName),
		})
		if err != nil {
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
			fmt.Printf("  [%s] %s %s\n",
				ts,
				aws.ToString(e.ResourceType),
				string(e.ResourceStatus),
			)
		}

		stackOut, err := cfnClient.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
			StackName: aws.String(stackName),
		})
		if err != nil {
			done, failed := isTerminal(cfntypes.StackStatusDeleteComplete)
			if done && !failed {
				return nil
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

		time.Sleep(5 * time.Second)
	}
}
```

**Step 2: Verify**

```bash
go build ./internal/provider/aws/cfn/...
```

---

### Task 6: Create `internal/provider/aws/cfn/destroy.go`

**Files:**
- Create: `internal/provider/aws/cfn/destroy.go`

Identical to `internal/cfn/destroy.go`:

```go
package cfn

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

func Destroy(ctx context.Context, cfg aws.Config, stackName string) error {
	cfnClient := cloudformation.NewFromConfig(cfg)

	_, err := cfnClient.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return fmt.Errorf("delete stack %s: %w", stackName, err)
	}

	return pollStackEvents(ctx, cfnClient, stackName, isDestroyTerminal)
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
```

**Step 2: Verify**

```bash
go build ./internal/provider/aws/cfn/...
```

---

### Task 7: Create `internal/provider/aws/state.go`

**Files:**
- Create: `internal/provider/aws/state.go`

Replaces `internal/state/manager.go`. Uses `provider.Deployment` instead of local struct.

```go
package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/havenapp/haven/internal/provider"
)

type S3StateStore struct {
	s3Client   *s3.Client
	bucketName string
}

func newS3StateStore(ctx context.Context, cfg awssdk.Config, accountID string) (*S3StateStore, error) {
	bucketName, err := ensureStateBucket(ctx, cfg, accountID)
	if err != nil {
		return nil, err
	}
	return &S3StateStore{
		s3Client:   s3.NewFromConfig(cfg),
		bucketName: bucketName,
	}, nil
}

func (s *S3StateStore) key(id string) string {
	return fmt.Sprintf("deployments/%s.json", id)
}

func (s *S3StateStore) Save(ctx context.Context, d provider.Deployment) error {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: awssdk.String(s.bucketName),
		Key:    awssdk.String(s.key(d.ID)),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (s *S3StateStore) Load(ctx context.Context, id string) (*provider.Deployment, error) {
	out, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: awssdk.String(s.bucketName),
		Key:    awssdk.String(s.key(id)),
	})
	if err != nil {
		return nil, fmt.Errorf("deployment %q not found: %w", id, err)
	}
	defer out.Body.Close()
	var d provider.Deployment
	if err := json.NewDecoder(out.Body).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *S3StateStore) List(ctx context.Context) ([]provider.Deployment, error) {
	out, err := s.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: awssdk.String(s.bucketName),
		Prefix: awssdk.String("deployments/"),
	})
	if err != nil {
		return nil, err
	}
	var deployments []provider.Deployment
	for _, obj := range out.Contents {
		id := strings.TrimSuffix(strings.TrimPrefix(awssdk.ToString(obj.Key), "deployments/"), ".json")
		d, err := s.Load(ctx, id)
		if err != nil {
			continue
		}
		deployments = append(deployments, *d)
	}
	return deployments, nil
}

func (s *S3StateStore) Delete(ctx context.Context, id string) error {
	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: awssdk.String(s.bucketName),
		Key:    awssdk.String(s.key(id)),
	})
	return err
}
```

**Step 2: Verify**

```bash
go build ./internal/provider/aws/...
```

---

### Task 8: Create `internal/provider/aws/provider.go`

**Files:**
- Create: `internal/provider/aws/provider.go`

The public entry point of the AWS provider. `New` returns both `provider.Provider` and `provider.StateStore` since both need the same AWS config.

```go
package aws

import (
	"context"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"

	"github.com/havenapp/haven/internal/provider"
	"github.com/havenapp/haven/internal/provider/aws/cfn"
)

// AWSProvider implements provider.Provider using CloudFormation.
type AWSProvider struct {
	cfg      awssdk.Config
	identity provider.Identity
}

// New creates an AWSProvider and S3StateStore, validating credentials eagerly.
func New(ctx context.Context) (*AWSProvider, *S3StateStore, error) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	id, err := getIdentity(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	store, err := newS3StateStore(ctx, cfg, id.AccountID)
	if err != nil {
		return nil, nil, err
	}

	return &AWSProvider{
		cfg: cfg,
		identity: provider.Identity{
			AccountID: id.AccountID,
			Region:    id.Region,
		},
	}, store, nil
}

func (p *AWSProvider) Identity(_ context.Context) (provider.Identity, error) {
	return p.identity, nil
}

func (p *AWSProvider) Deploy(ctx context.Context, input provider.DeployInput) (provider.DeployResult, error) {
	result, err := cfn.Deploy(ctx, p.cfg, cfn.DeployInput{
		StackName:    input.DeploymentID,
		Model:        input.Model,
		InstanceType: input.InstanceType,
		UserIP:       input.UserIP,
		APIKey:       input.APIKey,
	})
	if err != nil {
		return provider.DeployResult{}, err
	}
	return provider.DeployResult{
		ProviderRef: result.StackName,
		InstanceID:  result.InstanceID,
		PublicIP:    result.PublicIP,
	}, nil
}

func (p *AWSProvider) Destroy(ctx context.Context, providerRef string) error {
	return cfn.Destroy(ctx, p.cfg, providerRef)
}
```

**Step 2: Verify**

```bash
go build ./internal/provider/aws/...
```

---

### Task 9: Rewrite `internal/cli/root.go`

**Files:**
- Modify: `internal/cli/root.go`

Remove `init()`, use constructor pattern, add `--provider` flag, update Long description:

```go
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/provider"
	awsprovider "github.com/havenapp/haven/internal/provider/aws"
)

func NewRootCmd() *cobra.Command {
	var providerName string

	root := &cobra.Command{
		Use:   "haven",
		Short: "Deploy open-source LLM models to your own cloud",
		Long:  "Haven deploys LLM models to your cloud with one command.\nYour data never leaves your infrastructure.",
	}

	root.PersistentFlags().StringVar(&providerName, "provider", "aws", "Cloud provider to use (aws)")

	root.AddCommand(newDeployCmd(&providerName))
	root.AddCommand(newDestroyCmd(&providerName))
	root.AddCommand(newStatusCmd(&providerName))

	return root
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func buildProviderAndStore(ctx context.Context, name string) (provider.Provider, provider.StateStore, error) {
	switch name {
	case "aws":
		return awsprovider.New(ctx)
	default:
		return nil, nil, fmt.Errorf("unknown provider %q — available: aws", name)
	}
}
```

**Step 2: Verify**

```bash
go build ./internal/cli/...
```

---

### Task 10: Rewrite `internal/cli/deploy.go`

**Files:**
- Modify: `internal/cli/deploy.go`

Use `provider.Provider` + `provider.StateStore` interfaces. Remove direct aws/cfn/state imports.

```go
package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/havenapp/haven/internal/models"
	"github.com/havenapp/haven/internal/provider"
)

func newDeployCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:     "deploy <model>",
		Short:   "Deploy a model to your cloud",
		Example: "  haven deploy llama3.2:1b\n  haven deploy phi3:mini --provider aws",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(cmd.Context(), *providerName, args[0])
		},
	}
}

func runDeploy(ctx context.Context, providerName, modelName string) error {
	modelCfg, err := models.Lookup(modelName)
	if err != nil {
		return err
	}

	prov, store, err := buildProviderAndStore(ctx, providerName)
	if err != nil {
		return err
	}

	identity, err := prov.Identity(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Provider: %s  Account: %s  Region: %s\n", providerName, identity.AccountID, identity.Region)

	userIP, err := detectPublicIP()
	if err != nil {
		return fmt.Errorf("detect public IP: %w", err)
	}
	fmt.Printf("Restricting port 11434 to: %s\n\n", userIP)

	apiKey := generateAPIKey()
	deploymentID := generateDeploymentID()

	fmt.Printf("Deploying %s on %s (id: %s)...\n\n", modelName, modelCfg.InstanceType, deploymentID)

	result, err := prov.Deploy(ctx, provider.DeployInput{
		DeploymentID: deploymentID,
		Model:        modelCfg.OllamaTag,
		InstanceType: modelCfg.InstanceType,
		UserIP:       userIP + "/32",
		APIKey:       apiKey,
	})
	if err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	endpoint := fmt.Sprintf("http://%s:11434", result.PublicIP)
	fmt.Printf("\nInstance up at %s. Waiting for Ollama + model pull...\n", result.PublicIP)

	if err := waitForOllama(ctx, endpoint, modelName, apiKey); err != nil {
		return fmt.Errorf("waiting for Ollama: %w", err)
	}

	deployment := provider.Deployment{
		ID:           deploymentID,
		Provider:     providerName,
		ProviderRef:  result.ProviderRef,
		CreatedAt:    time.Now().UTC(),
		Region:       identity.Region,
		Model:        modelName,
		InstanceType: modelCfg.InstanceType,
		InstanceID:   result.InstanceID,
		PublicIP:     result.PublicIP,
		Endpoint:     endpoint + "/v1",
		APIKey:       apiKey,
	}

	if err := store.Save(ctx, deployment); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	fmt.Printf("\nDeployment ready!\n")
	fmt.Printf("  Endpoint : %s\n", deployment.Endpoint)
	fmt.Printf("  API Key  : %s\n", deployment.APIKey)
	fmt.Printf("  ID       : %s\n\n", deployment.ID)
	fmt.Printf("Test:\n")
	fmt.Printf("  curl %s/chat/completions \\\n", deployment.Endpoint)
	fmt.Printf("    -H 'Authorization: Bearer %s' \\\n", deployment.APIKey)
	fmt.Printf("    -H 'Content-Type: application/json' \\\n")
	fmt.Printf("    -d '{\"model\":\"%s\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}]}'\n", modelName)
	return nil
}

func detectPublicIP() (string, error) {
	resp, err := http.Get("https://checkip.amazonaws.com/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func waitForOllama(ctx context.Context, endpoint, model, apiKey string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	deadline := time.Now().Add(15 * time.Minute)
	modelBase := strings.SplitN(model, ":", 2)[0]

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/api/tags", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 && strings.Contains(string(body), modelBase) {
				fmt.Println(" ready!")
				return nil
			}
		}
		fmt.Print(".")
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("timed out after 15 minutes")
}

func generateAPIKey() string {
	b := make([]byte, 18)
	rand.Read(b)
	return "sk-haven-" + hex.EncodeToString(b)
}

func generateDeploymentID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return "haven-" + hex.EncodeToString(b)
}
```

**Step 2: Verify**

```bash
go build ./internal/cli/...
```

---

### Task 11: Rewrite `internal/cli/destroy.go`

**Files:**
- Modify: `internal/cli/destroy.go`

```go
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newDestroyCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:     "destroy <deployment-id>",
		Short:   "Destroy a deployment and release all cloud resources",
		Example: "  haven destroy haven-a1b2c3d4",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDestroy(cmd.Context(), *providerName, args[0])
		},
	}
}

func runDestroy(ctx context.Context, providerName, deploymentID string) error {
	prov, store, err := buildProviderAndStore(ctx, providerName)
	if err != nil {
		return err
	}

	deployment, err := store.Load(ctx, deploymentID)
	if err != nil {
		return err
	}

	fmt.Printf("Destroying %s (%s on %s)...\n\n", deployment.ID, deployment.Model, deployment.InstanceType)

	if err := prov.Destroy(ctx, deployment.ProviderRef); err != nil {
		return fmt.Errorf("destroy: %w", err)
	}

	if err := store.Delete(ctx, deploymentID); err != nil {
		fmt.Printf("Warning: failed to delete state for %s: %v\n", deploymentID, err)
	}

	fmt.Printf("\nDestroyed %s. All resources released.\n", deploymentID)
	return nil
}
```

**Step 2: Verify**

```bash
go build ./internal/cli/...
```

---

### Task 12: Rewrite `internal/cli/status.go`

**Files:**
- Modify: `internal/cli/status.go`

Adds `CLOUD` column to output:

```go
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCmd(providerName *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "List active deployments",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd.Context(), *providerName)
		},
	}
}

func runStatus(ctx context.Context, providerName string) error {
	_, store, err := buildProviderAndStore(ctx, providerName)
	if err != nil {
		return err
	}

	deployments, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("list deployments: %w", err)
	}

	if len(deployments) == 0 {
		fmt.Println("No active deployments.")
		return nil
	}

	fmt.Printf("%-20s  %-6s  %-14s  %-12s  %s\n", "ID", "CLOUD", "MODEL", "INSTANCE", "ENDPOINT")
	fmt.Printf("%-20s  %-6s  %-14s  %-12s  %s\n", "--------------------", "------", "--------------", "------------", "--------")
	for _, d := range deployments {
		fmt.Printf("%-20s  %-6s  %-14s  %-12s  %s\n", d.ID, d.Provider, d.Model, d.InstanceType, d.Endpoint)
	}
	return nil
}
```

**Step 2: Verify**

```bash
go build ./internal/cli/...
```

---

### Task 13: Update `cmd/haven/main.go`

**Files:**
- Modify: `cmd/haven/main.go`

```go
package main

import (
	"fmt"
	"os"

	"github.com/havenapp/haven/internal/cli"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	return cli.NewRootCmd().Execute()
}
```

**Step 2: Verify full build**

```bash
go build ./...
```

Expected: no errors.

---

### Task 14: Delete old packages and final verification

**Step 1: Delete old directories**

```bash
rm -rf internal/aws internal/cfn internal/state
```

**Step 2: Full build**

```bash
go build ./...
```

Expected: success with no errors.

**Step 3: Verify help output looks correct**

```bash
go run ./cmd/haven/ --help
go run ./cmd/haven/ deploy --help
go run ./cmd/haven/ destroy --help
go run ./cmd/haven/ status --help
```

Confirm:
- No mention of "AWS" in top-level help
- `--provider` flag is visible in all commands
- `deploy --help` shows `--provider aws` in example

**Step 4: Tidy deps**

```bash
go mod tidy
```

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor: introduce provider/statestore interfaces for multi-cloud support

Move AWS implementation to internal/provider/aws/, define Provider and
StateStore interfaces in internal/provider/. Add --provider flag to CLI.
Remove AWS-specific language from help text. Fix hardcoded t3.large in
CloudFormation template."
```
