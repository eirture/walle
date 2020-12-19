package changelog

import (
	"fmt"
	"testing"
)

func TestReadChangelogFile(t *testing.T) {
	content, err := GenerateChangelog("v1.0.5", "\n- test\n- abc\n\n", "/Users/liujie/repo/ticket-backend/CHANGELOG.md")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(content)
}
