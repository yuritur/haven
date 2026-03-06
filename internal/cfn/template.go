package cfn

import (
	"encoding/json"
	"strings"
)

type TemplateInput struct {
	UserIP string
	APIKey string
	Model  string
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
					"InstanceType":       "t3.large",
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
