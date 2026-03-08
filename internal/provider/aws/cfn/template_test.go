package cfn

import (
	"encoding/json"
	"testing"

	"github.com/havenapp/haven/internal/models"
)

func testInput() TemplateInput {
	return TemplateInput{
		UserIP:       "203.0.113.1/32",
		APIKey:       "sk-haven-test",
		Runtime:      models.RuntimeOllama,
		ModelTag:     "llama3.2:1b",
		InstanceType: "t3.large",
		TLSCert:      "FAKE_CERT_PEM",
		TLSKey:       "FAKE_KEY_PEM",
	}
}

func parseTemplate(t *testing.T, jsonStr string) map[string]interface{} {
	t.Helper()
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	return parsed
}

func TestGenerateTemplate_ValidJSON(t *testing.T) {
	out, err := GenerateTemplate(testInput())
	if err != nil {
		t.Fatalf("GenerateTemplate returned error: %v", err)
	}
	parseTemplate(t, out)
}

func TestGenerateTemplate_SecurityGroup(t *testing.T) {
	input := testInput()
	out, err := GenerateTemplate(input)
	if err != nil {
		t.Fatalf("GenerateTemplate returned error: %v", err)
	}
	parsed := parseTemplate(t, out)

	resources, ok := parsed["Resources"].(map[string]interface{})
	if !ok {
		t.Fatal("Resources not found or not an object")
	}
	sg, ok := resources["HavenSG"].(map[string]interface{})
	if !ok {
		t.Fatal("HavenSG not found or not an object")
	}
	props, ok := sg["Properties"].(map[string]interface{})
	if !ok {
		t.Fatal("HavenSG Properties not found or not an object")
	}
	ingress, ok := props["SecurityGroupIngress"].([]interface{})
	if !ok {
		t.Fatal("SecurityGroupIngress not found or not an array")
	}
	if len(ingress) == 0 {
		t.Fatal("no ingress rules found")
	}
	rule, ok := ingress[0].(map[string]interface{})
	if !ok {
		t.Fatal("first ingress rule is not an object")
	}
	cidr, ok := rule["CidrIp"].(string)
	if !ok {
		t.Fatal("CidrIp not found or not a string")
	}
	if cidr != input.UserIP {
		t.Errorf("CidrIp = %q, want %q", cidr, input.UserIP)
	}
}

func TestGenerateTemplate_InstanceType(t *testing.T) {
	input := testInput()
	input.InstanceType = "t3.xlarge"
	out, err := GenerateTemplate(input)
	if err != nil {
		t.Fatalf("GenerateTemplate returned error: %v", err)
	}
	parsed := parseTemplate(t, out)

	resources, ok := parsed["Resources"].(map[string]interface{})
	if !ok {
		t.Fatal("Resources not found or not an object")
	}
	instance, ok := resources["HavenInstance"].(map[string]interface{})
	if !ok {
		t.Fatal("HavenInstance not found or not an object")
	}
	props, ok := instance["Properties"].(map[string]interface{})
	if !ok {
		t.Fatal("HavenInstance Properties not found or not an object")
	}
	it, ok := props["InstanceType"].(string)
	if !ok {
		t.Fatal("InstanceType not found or not a string")
	}
	if it != "t3.xlarge" {
		t.Errorf("InstanceType = %q, want %q", it, "t3.xlarge")
	}
}

func TestGenerateTemplate_Outputs(t *testing.T) {
	out, err := GenerateTemplate(testInput())
	if err != nil {
		t.Fatalf("GenerateTemplate returned error: %v", err)
	}
	parsed := parseTemplate(t, out)

	outputs, ok := parsed["Outputs"].(map[string]interface{})
	if !ok {
		t.Fatal("Outputs not found or not an object")
	}
	for _, key := range []string{"InstanceId", "PublicIP"} {
		if _, ok := outputs[key]; !ok {
			t.Errorf("output %q not found", key)
		}
	}
}
