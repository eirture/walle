package releasenote

import (
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
				"without release",
				"feat(cmd): test cmd EE-1",
			},
			"_New Features:_\n- first commit @liujie\n- cmd: test cmd EE-1\n\n**Bug Fix:**\n- api: test bugfix\n",
		},
	}

	for i, tc := range testcases {
		result := GenerateReleaseNotes(tc.values)
		if result != tc.expected {
			t.Errorf("failed to assert equeal case %d of \n%s and \n%s", i, result, tc.expected)
		}
	}
}
