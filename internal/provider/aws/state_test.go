package aws

import "testing"

func TestKey(t *testing.T) {
	store := S3StateStore{}
	got := store.key("haven-abc123")
	want := "deployments/haven-abc123.json"
	if got != want {
		t.Errorf("key(\"haven-abc123\") = %q, want %q", got, want)
	}
}
