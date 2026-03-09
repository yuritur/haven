package quota

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
)

const serviceCode = "ec2"

type QuotaStatus struct {
	CurrentVCPUs  float64
	RequiredVCPUs int
	Sufficient    bool
	QuotaCode     string
}

type QuotaRequest struct {
	RequestID    string    `json:"request_id"`
	QuotaCode    string    `json:"quota_code"`
	Status       string    `json:"status"`
	DesiredVCPUs float64   `json:"desired_vcpus"`
	CreatedAt    time.Time `json:"created_at"`
	InstanceType string    `json:"instance_type"`
}

var familyToQuotaCode = map[string]string{
	"g4dn": "L-DB2BBE81",
	"g5":   "L-DB2BBE81",
	"g5g":  "L-DB2BBE81",
	"g6":   "L-DB2BBE81",
	"p3":   "L-417A185B",
	"p4":   "L-417A185B",
	"p5":   "L-417A185B",
}

var instanceVCPUs = map[string]int{
	"g4dn.xlarge": 4,
	"g5.xlarge":   4,
	"g5.2xlarge":  8,
	"g5g.xlarge":  4,
	"g6.xlarge":   4,
	"p3.2xlarge":  8,
}

func QuotaCodeForInstance(instanceType string) (string, error) {
	family := instanceFamily(instanceType)
	code, ok := familyToQuotaCode[family]
	if !ok {
		return "", fmt.Errorf("unknown GPU instance family %q", family)
	}
	return code, nil
}

func VCPUsForInstance(instanceType string) (int, error) {
	v, ok := instanceVCPUs[instanceType]
	if !ok {
		return 0, fmt.Errorf("unknown vCPU count for instance type %q", instanceType)
	}
	return v, nil
}

func instanceFamily(instanceType string) string {
	if idx := strings.Index(instanceType, "."); idx > 0 {
		return instanceType[:idx]
	}
	return instanceType
}

func CheckQuota(ctx context.Context, cfg aws.Config, instanceType string) (*QuotaStatus, error) {
	quotaCode, err := QuotaCodeForInstance(instanceType)
	if err != nil {
		return nil, err
	}
	vcpus, err := VCPUsForInstance(instanceType)
	if err != nil {
		return nil, err
	}

	client := servicequotas.NewFromConfig(cfg)
	out, err := client.GetServiceQuota(ctx, &servicequotas.GetServiceQuotaInput{
		ServiceCode: aws.String(serviceCode),
		QuotaCode:   aws.String(quotaCode),
	})
	if err != nil {
		return nil, fmt.Errorf("get service quota %s: %w", quotaCode, err)
	}

	current := aws.ToFloat64(out.Quota.Value)
	return &QuotaStatus{
		CurrentVCPUs:  current,
		RequiredVCPUs: vcpus,
		Sufficient:    current >= float64(vcpus),
		QuotaCode:     quotaCode,
	}, nil
}

func RequestIncrease(ctx context.Context, cfg aws.Config, quotaCode string, desiredValue float64) (*QuotaRequest, error) {
	client := servicequotas.NewFromConfig(cfg)
	out, err := client.RequestServiceQuotaIncrease(ctx, &servicequotas.RequestServiceQuotaIncreaseInput{
		ServiceCode:  aws.String(serviceCode),
		QuotaCode:    aws.String(quotaCode),
		DesiredValue: aws.Float64(desiredValue),
	})
	if err != nil {
		return nil, fmt.Errorf("request quota increase for %s: %w", quotaCode, err)
	}

	return &QuotaRequest{
		RequestID:    aws.ToString(out.RequestedQuota.Id),
		QuotaCode:    quotaCode,
		Status:       string(out.RequestedQuota.Status),
		DesiredVCPUs: desiredValue,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

func GetRequestStatus(ctx context.Context, cfg aws.Config, requestID string) (string, error) {
	client := servicequotas.NewFromConfig(cfg)
	out, err := client.GetRequestedServiceQuotaChange(ctx, &servicequotas.GetRequestedServiceQuotaChangeInput{
		RequestId: aws.String(requestID),
	})
	if err != nil {
		return "", fmt.Errorf("get quota request status %s: %w", requestID, err)
	}
	return string(out.RequestedQuota.Status), nil
}
