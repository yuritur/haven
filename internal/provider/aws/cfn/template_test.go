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

func TestGenerateTemplate_ValidJSON(t *testing.T) {
	out, err := GenerateTemplate(testInput())
	if err != nil {
		t.Fatalf("GenerateTemplate returned error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestGenerateTemplate_Resources(t *testing.T) {
	out, err := GenerateTemplate(testInput())
	if err != nil {
		t.Fatalf("GenerateTemplate returned error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	resources, ok := parsed["Resources"].(map[string]interface{})
	if !ok {
		t.Fatal("Resources not found or not an object")
	}
	expected := []string{
		"HavenVPC", "HavenSubnet", "HavenIGW", "HavenVPCGWAttachment",
		"HavenRouteTable", "HavenRoute", "HavenSubnetRTAssoc",
		"HavenSG", "HavenRole", "HavenInstanceProfile",
		"HavenInstance", "HavenEIP", "HavenEIPAssoc",
	}
	for _, name := range expected {
		if _, ok := resources[name]; !ok {
			t.Errorf("resource %q not found", name)
		}
	}
	if len(resources) != 13 {
		t.Errorf("resource count = %d, want 13", len(resources))
	}
}

func TestGenerateTemplate_SecurityGroup(t *testing.T) {
	input := testInput()
	out, err := GenerateTemplate(input)
	if err != nil {
		t.Fatalf("GenerateTemplate returned error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	resources := parsed["Resources"].(map[string]interface{})
	sg := resources["HavenSG"].(map[string]interface{})
	props := sg["Properties"].(map[string]interface{})
	ingress := props["SecurityGroupIngress"].([]interface{})
	if len(ingress) == 0 {
		t.Fatal("no ingress rules found")
	}
	rule := ingress[0].(map[string]interface{})
	cidr, _ := rule["CidrIp"].(string)
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
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	resources := parsed["Resources"].(map[string]interface{})
	instance := resources["HavenInstance"].(map[string]interface{})
	props := instance["Properties"].(map[string]interface{})
	it, _ := props["InstanceType"].(string)
	if it != "t3.xlarge" {
		t.Errorf("InstanceType = %q, want %q", it, "t3.xlarge")
	}
}

func TestGenerateTemplate_Outputs(t *testing.T) {
	out, err := GenerateTemplate(testInput())
	if err != nil {
		t.Fatalf("GenerateTemplate returned error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

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
