package releasenote

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"walle/pkg/gitlab"
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

func GetReleaseNotesByTag(client gitlab.Client, project, tagName, ref string) (
	tagExists bool, releaseNotes string, err error,
) {
	tags, err := client.ListTags(project)
	if err != nil {
		return
	}
	var sinceAt, untilAt *time.Time

	for i := 0; i < len(tags); i++ {
		tag := tags[i]
		if tag.Name == tagName {
			tagExists = true
			untilAt = &tag.Commit.CreatedAt
			continue
		}
		sinceAt = &tag.Commit.CreatedAt
		break
	}

	commits, err := client.ListCommits(project, ref, sinceAt, untilAt)
	if err != nil {
		logrus.Errorf("An error occurred while list commits. %v", err)
		return
	}
	if len(commits) > 0 {
		// the first commit belong to the tag before this
		commits = commits[1:]
	}

	notes := getNotes(commits)
	releaseNotes = GenerateReleaseNotes(notes)
	return
}

func getNotes(commits []*gitlab.Commit) (filtered []string) {
	for _, commit := range commits {
		res := notesForCommit(commit.Message)
		if res == "" {
			continue
		}
		filtered = append(filtered, res)
	}
	return
}

func notesForCommit(commitMessage string) string {
	regex := regexp.MustCompile(`\n\n(?P<notes>[^\n]*)\n\nSee merge request`)
	match := regex.FindStringSubmatch(commitMessage)
	if match == nil {
		return ""
	}

	for i, name := range regex.SubexpNames() {
		if i != 0 && name != "" {
			return match[i]
		}
	}
	return ""
}
