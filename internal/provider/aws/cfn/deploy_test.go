package cfn

import (
	"testing"

	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

func TestIsDeployTerminal(t *testing.T) {
	cases := []struct {
		status     cfntypes.StackStatus
		wantDone   bool
		wantFailed bool
	}{
		{cfntypes.StackStatusCreateComplete, true, false},
		{cfntypes.StackStatusCreateFailed, true, true},
		{cfntypes.StackStatusRollbackComplete, true, true},
		{cfntypes.StackStatusRollbackFailed, true, true},
		{cfntypes.StackStatusCreateInProgress, false, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			done, failed := isDeployTerminal(tc.status)
			if done != tc.wantDone {
				t.Errorf("done = %v, want %v", done, tc.wantDone)
			}
			if failed != tc.wantFailed {
				t.Errorf("failed = %v, want %v", failed, tc.wantFailed)
			}
		})
	}
}
