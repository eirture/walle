package releasenote

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"walle/pkg/gitlab"
	"walle/pkg/utils"
)

const (
	titleChanges       = "_Changes:_"
	titleBugFix        = "**Bug Fix:**"
	titleNewFeature    = "_New Features:_"
	titleDocumentation = "Documentation:"
	titleOther         = "Other:"
)

var (
	tagMatcherRe = regexp.MustCompile(`^([^( ]+)\((.*)\)$`)
	kinds        = map[string]string{
		"feat":     titleNewFeature,
		"fix":      titleBugFix,
		"refactor": titleChanges,
		"docs":     titleDocumentation,
	}
	defaultKind = titleOther
	sortedKinds = []string{
		titleBugFix,
		titleNewFeature,
		titleChanges,
		titleDocumentation,
		titleOther,
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
		kind, ok := kinds[tag]
		if !ok {
			kind = defaultKind
			if strings.Contains(tag, " ") {
				// do nothing for this item
				summary = i
			}
		}

		releases[kind] = append(releases[kind], summary)
	}

	var values []string
	for _, k := range sortedKinds {
		v, ok := releases[k]
		if !ok {
			continue
		}
		values = append(values, fmt.Sprintf("%s\n- %s\n", k, strings.Join(v, "\n- ")))
	}
	return strings.Join(values, "\n")
}

func ReleaseNotesFromMR(mrs []gitlab.MergeRequest, condition func(mr *gitlab.MergeRequest) bool) string {
	if condition == nil {
		condition = func(mr *gitlab.MergeRequest) bool { return true }
	}
	var titles []string
	for _, mr := range mrs {
		if !condition(&mr) {
			continue
		}
		titles = append(titles, fmt.Sprintf(
			"%s ([!%d](%s)) @%s",
			mr.Title,
			mr.IID,
			mr.WebURL,
			mr.Author.Username,
		))
	}

	return GenerateReleaseNotes(titles)
}

func ChangelogFromMR(client gitlab.Client, project, tagName string, branches []string) (exists bool, changelog string, err error) {
	tags, err := client.ListTags(project)
	if err != nil {
		return
	}
	afterAt, beforeAt := time.Unix(0, 0), time.Now()

	var afterAtSHA, beforeAtSHA string
	for i := 0; i < len(tags); i++ {
		tag := tags[i]
		if tag.Name == tagName {
			exists = true
			beforeAt = tag.Commit.CreatedAt
			beforeAtSHA = tag.Commit.ID
			continue
		}
		afterAtSHA = tag.Commit.ID
		afterAt = tag.Commit.CreatedAt
		break
	}

	mrs, err := client.ListMergeRequests(project, afterAt)
	if err != nil {
		return
	}

	afterCond := func(mr *gitlab.MergeRequest) bool {
		return mr.MergeCommitSHA != afterAtSHA
	}

	beforeCond := func(mr *gitlab.MergeRequest) bool {
		return beforeAt.After(mr.MergedAt) || beforeAtSHA == "" || beforeAtSHA == mr.MergeCommitSHA
	}

	branchCond := func(mr *gitlab.MergeRequest) bool {
		return len(branches) == 0 || utils.InStringArray(mr.TargetBranch, branches)
	}

	condition := func(mr *gitlab.MergeRequest) bool {
		return afterCond(mr) && beforeCond(mr) && branchCond(mr)
	}

	changelog = ReleaseNotesFromMR(mrs, condition)
	return
}
