package releasenote

import (
	"fmt"
	"testing"
)

func TestGenerateReleaseNotes(t *testing.T) {
	testcases := []struct {
		values   []string
		expected string
	}{
		{
			[]string{
				"feat: first commit @liujie",
				"fix(api): test bugfix",
				"no kind MR",
				"feat(cmd): test cmd EE-1",
			},
			"**Bug Fix:**\n- api: test bugfix\n\n_New Features:_\n- first commit @liujie\n- cmd: test cmd EE-1\n\nOther:\n- no kind MR\n",
		},
	}

	for i, tc := range testcases {
		result := joinNotes(tc.values)
		fmt.Println(result)
		if result != tc.expected {
			t.Errorf("failed to assert equeal case %d of \n%s and \n%s", i, result, tc.expected)
		}
	}
}
