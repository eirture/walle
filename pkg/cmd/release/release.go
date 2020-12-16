package release

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"walle/pkg/config"
	"walle/pkg/context"
	"walle/pkg/gitlab"
	"walle/pkg/releasenote"
)

func NewReleaseCmd(ctx *context.Context) *cobra.Command {
	opts := &releaseOptions{
		client: ctx.GitLabClient,
		cfg:    ctx.Config,
		logger: ctx.Logger,
	}
	cmd := &cobra.Command{
		Use:   "release",
		Short: "release a new version ",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.project = ctx.Project
			if err := opts.Run(cmd, args); err != nil {
				ctx.Logger.Error(err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.targetBranch, "branch", "b", "master", "the target branch name of merge request")
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "the tag name which will be released (required)")
	_ = cmd.MarkFlagRequired("tag")
	return cmd
}

type releaseOptions struct {
	client gitlab.Client
	cfg    *config.Config
	logger *logrus.Entry

	targetBranch string
	tag          string
	project      string
}

func (o *releaseOptions) Run(cmd *cobra.Command, args []string) error {
	tags, err := o.client.ListTags(o.project)
	if err != nil {
		return err
	}
	afterAt := time.Unix(0, 0)
	if len(tags) > 0 {
		afterAt = tags[0].Commit.CreatedAt
	}

	mrs, err := o.client.ListMergeRequests(o.project, afterAt)
	if err != nil {
		return err
	}
	var titles []string
	for _, mr := range mrs {
		title := fmt.Sprintf("%s ([#%d](%s)) @%s",
			mr.Title,
			mr.IID,
			mr.WebURL,
			mr.Author.Username,
		)
		fmt.Println(title)
		titles = append(titles, title)
	}
	fmt.Println(releasenote.GenerateReleaseNotes(titles))
	return nil
}
