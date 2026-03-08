package aws

import "testing"

func TestStateBucketName(t *testing.T) {
	got := stateBucketName("123456789")
	want := "haven-state-123456789"
	if got != want {
		t.Errorf("stateBucketName(\"123456789\") = %q, want %q", got, want)
	}
}
