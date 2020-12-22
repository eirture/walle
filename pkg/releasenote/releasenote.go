package releasenote

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"walle/pkg/gitlab"
	"walle/pkg/utils"
)

const (
	titleChanges       = "_Changes:_"
	titleBugFix        = "**Bug Fix:**"
	titleNewFeature    = "_New Features:_"
	titleDocumentation = "Documentation:"
	titleOther         = "Other:"

	labelReleaseNoteNone = "release-note-none"
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

var noteExclusionFilters = []*regexp.Regexp{
	// 'none','n/a','na' case insensitive with optional trailing
	// whitespace, wrapped in ``` with/without release-note identifier
	// the 'none','n/a','na' can also optionally be wrapped in quotes ' or "
	regexp.MustCompile("(?i)```release-note[s]?\\s*('|\")?(none|n/a|na)?('|\")?\\s*```"),

	// simple '/release-note-none' tag
	regexp.MustCompile("/release-note-none"),
}

// MatchesExcludeFilter returns true if the string matches an excluded release note
func MatchesExcludeFilter(msg string) bool {
	return matchesFilter(msg, noteExclusionFilters)
}

func matchesFilter(msg string, filters []*regexp.Regexp) bool {
	for _, filter := range filters {
		if filter.MatchString(msg) {
			return true
		}
	}
	return false
}

func joinNotes(items []string) string {
	releases := make(map[string][]string)
	for _, i := range items {
		is := strings.SplitN(i, ":", 2)
		if len(is) != 2 {
			is = []string{"", i}
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

func generateReleaseNotes(mrs []*gitlab.MergeRequest, condition func(mr *gitlab.MergeRequest) bool) string {
	if condition == nil {
		condition = func(mr *gitlab.MergeRequest) bool { return true }
	}
	var titles []string
	for _, mr := range mrs {
		if !condition(mr) {
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

	return joinNotes(titles)
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

	var mrs []*gitlab.MergeRequest
	for _, commit := range commits {
		iid := mrNumForCommitFromMessage(commit.Message)
		if iid == 0 {
			continue
		}
		mr, err := client.GetMergeRequest(project, iid)
		if err != nil {
			logrus.Warnf("an error occurred while get merge request %d. %s", iid, err)
			continue
		}
		mrs = append(mrs, mr)
	}

	condition := func(mr *gitlab.MergeRequest) bool {
		// do not have the label `release-note-none`
		exclude := MatchesExcludeFilter(mr.Description) || utils.InStringArray(labelReleaseNoteNone, mr.Labels)
		return !exclude
	}
	releaseNotes = generateReleaseNotes(mrs, condition)
	return
}

func mrNumForCommitFromMessage(commitMessage string) (mr int) {
	regex := regexp.MustCompile(`\n\nSee merge request .+!(\d+)$`)
	match := regex.FindStringSubmatch(commitMessage)
	if match == nil || len(match) < 2 {
		return 0
	}
	mr, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return
}
