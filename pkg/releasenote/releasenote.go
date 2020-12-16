package releasenote

import (
	"fmt"
	"regexp"
	"strings"

	"walle/pkg/gitlab"
)

const (
	titleChanges    = "_Changes:_"
	titleBugFix     = "**Bug Fix:**"
	titleNewFeature = "_New Features:_"
)

var (
	tagMatcherRe = regexp.MustCompile("^([^( ]+)\\((.*)\\)$")
	scopes       = map[string]string{
		"feat":     titleNewFeature,
		"fix":      titleBugFix,
		"refactor": titleChanges,
	}
)

func GenerateReleaseNotes(items []string) string {
	releases := make(map[string][]string)
	for _, i := range items {
		is := strings.SplitN(i, ":", 2)
		if len(is) != 2 {
			continue
		}
		tag, summary := strings.Trim(is[0], " "), strings.Trim(is[1], " ")
		if strings.Contains(tag, "(") {
			potentialMatch := tagMatcherRe.FindStringSubmatch(tag)
			if len(potentialMatch) == 3 {
				tag = potentialMatch[1]
				scope := potentialMatch[2]
				if scope != "" && scope != "*" {
					summary = fmt.Sprintf("%s: %s", scope, summary)
				}
			}
		}
		if strings.Contains(tag, " ") {
			continue
		}
		key, ok := scopes[tag]
		if !ok {
			continue
		}

		releases[key] = append(releases[key], summary)
	}

	var values []string
	for k, v := range releases {
		values = append(values, fmt.Sprintf("%s\n- %s\n", k, strings.Join(v, "\n- ")))
	}
	return strings.Join(values, "\n")
}

func ReleaseNotesFromMR(mrs []gitlab.MergeRequest, condition func(v string) bool) string {
	if condition == nil {
		condition = func(v string) bool { return true }
	}
	var titles []string
	for _, mr := range mrs {
		if !condition(mr.TargetBranch) {
			continue
		}
		titles = append(titles, fmt.Sprintf(
			"%s ([#%d](%s)) @%s",
			mr.Title,
			mr.IID,
			mr.WebURL,
			mr.Author.Username,
		))
	}

	return GenerateReleaseNotes(titles)
}
