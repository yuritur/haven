package cfn

import (
	"testing"

	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

func TestIsDestroyTerminal(t *testing.T) {
	cases := []struct {
		status     cfntypes.StackStatus
		wantDone   bool
		wantFailed bool
	}{
		{cfntypes.StackStatusDeleteComplete, true, false},
		{cfntypes.StackStatusDeleteFailed, true, true},
		{cfntypes.StackStatusDeleteInProgress, false, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			done, failed := isDestroyTerminal(tc.status)
			if done != tc.wantDone {
				t.Errorf("done = %v, want %v", done, tc.wantDone)
			}
			if failed != tc.wantFailed {
				t.Errorf("failed = %v, want %v", failed, tc.wantFailed)
			}
		})
	}
}
