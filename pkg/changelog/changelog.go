package changelog

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"walle/pkg/utils"
)

func GenerateChangelog(tagName, content string, filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer utils.CloseSilently(f)

	contentLines := strings.Split(content, "\n")
	var lines []string
	skipFlag := false
	tagExistsFlag := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# ") {
			tag := strings.SplitN(line[2:], " ", 2)[0]
			skipFlag = tag == tagName
			tagExistsFlag = tagExistsFlag || skipFlag
			if skipFlag {
				lines = append(lines, line)
				lines = append(lines, contentLines...)
				continue
			}
		}
		if skipFlag {
			continue
		}
		lines = append(lines, line)
	}

	if !tagExistsFlag {
		result := []string{fmt.Sprintf("# %s (%s)", tagName, time.Now().Format("2006-01-02"))}
		result = append(result, contentLines...)
		lines = append(result, lines...)
	}

	return strings.Join(lines, "\n"), nil
}
