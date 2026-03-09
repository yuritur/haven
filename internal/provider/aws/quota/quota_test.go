package quota

import (
	"testing"
)

func TestInstanceFamily(t *testing.T) {
	cases := []struct {
		instanceType string
		want         string
	}{
		{"g5.xlarge", "g5"},
		{"g4dn.xlarge", "g4dn"},
		{"p3.2xlarge", "p3"},
		{"t3.large", "t3"},
		{"m5.large", "m5"},
		{"g5g.xlarge", "g5g"},
		{"g6.xlarge", "g6"},
		{"nodot", "nodot"},
	}
	for _, tc := range cases {
		t.Run(tc.instanceType, func(t *testing.T) {
			got := instanceFamily(tc.instanceType)
			if got != tc.want {
				t.Errorf("instanceFamily(%q) = %q, want %q", tc.instanceType, got, tc.want)
			}
		})
	}
}

func TestQuotaCodeForInstance(t *testing.T) {
	cases := []struct {
		instanceType string
		wantCode     string
		wantErr      bool
	}{
		{"g5.xlarge", "L-DB2E81BA", false},
		{"g4dn.xlarge", "L-DB2E81BA", false},
		{"g5g.xlarge", "L-DB2E81BA", false},
		{"g6.xlarge", "L-DB2E81BA", false},
		{"p3.2xlarge", "L-417A185B", false},
		{"t3.large", "", true},
		{"m5.large", "", true},
		{"unknown", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.instanceType, func(t *testing.T) {
			code, err := QuotaCodeForInstance(tc.instanceType)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.instanceType)
				}
				return
			}
			if err != nil {
				t.Fatalf("QuotaCodeForInstance(%q) returned error: %v", tc.instanceType, err)
			}
			if code != tc.wantCode {
				t.Errorf("QuotaCodeForInstance(%q) = %q, want %q", tc.instanceType, code, tc.wantCode)
			}
		})
	}
}

func TestVCPUsForInstance(t *testing.T) {
	cases := []struct {
		instanceType string
		wantVCPUs    int
		wantErr      bool
	}{
		{"g4dn.xlarge", 4, false},
		{"g5.xlarge", 4, false},
		{"g5.2xlarge", 8, false},
		{"g5g.xlarge", 4, false},
		{"g6.xlarge", 4, false},
		{"p3.2xlarge", 8, false},
		{"t3.large", 0, true},
		{"m5.large", 0, true},
		{"nonexistent.type", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.instanceType, func(t *testing.T) {
			vcpus, err := VCPUsForInstance(tc.instanceType)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.instanceType)
				}
				return
			}
			if err != nil {
				t.Fatalf("VCPUsForInstance(%q) returned error: %v", tc.instanceType, err)
			}
			if vcpus != tc.wantVCPUs {
				t.Errorf("VCPUsForInstance(%q) = %d, want %d", tc.instanceType, vcpus, tc.wantVCPUs)
			}
		})
	}
}
