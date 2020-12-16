package changelog

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"walle/pkg/context"
	"walle/pkg/gitlab"
	"walle/pkg/releasenote"
	"walle/pkg/utils"
)

func NewCmdChangelog(ctx *context.Context) *cobra.Command {
	opts := Options{
		GitLabClient: ctx.GitLabClient,
	}

	cmd := &cobra.Command{
		Use:   "changelog",
		Short: "generate changelog",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.project = ctx.Project
			err := opts.Run()
			return err
		},
	}
	cmd.Flags().StringArrayVarP(&opts.branches, "branch", "b", []string{"master"}, "the branch name")

	return cmd
}

type Options struct {
	GitLabClient gitlab.Client

	project  string
	branches []string
}

func (o *Options) Run() error {
	tags, err := o.GitLabClient.ListTags(o.project)
	if err != nil {
		return err
	}
	afterAt := time.Unix(0, 0)
	if len(tags) > 0 {
		afterAt = tags[0].Commit.CreatedAt
	}

	mrs, err := o.GitLabClient.ListMergeRequests(o.project, afterAt)

	condition := func(_ string) bool { return true }
	if len(o.branches) > 0 {
		condition = func(v string) bool {
			return utils.InStringArray(v, o.branches)
		}
	}

	result := releasenote.ReleaseNotesFromMR(mrs, condition)
	if result != "" {
		fmt.Print(result)
	}
	return nil
}
