package changelog

import (
	"bufio"
	"fmt"
	"strings"
	"time"
)

var (
	DateLayout = "2006-01-02"
)

func GenerateChangelog(tagName, content string, originContent string, date time.Time) (string, error) {

	contentLines := strings.Split(content, "\n")
	var lines []string
	skipFlag := false
	tagExistsFlag := false

	scanner := bufio.NewScanner(strings.NewReader(originContent))
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
		result := []string{fmt.Sprintf("# %s (%s)", tagName, date.Format(DateLayout))}
		result = append(result, contentLines...)
		lines = append(result, lines...)
	}

	return strings.Join(lines, "\n"), nil
}
