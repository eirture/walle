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
	"walle/pkg/utils"
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
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&opts.branches, "branch", "b", []string{}, "the target branch name of merge request")
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "The name of a tag (required)")
	cmd.Flags().StringVarP(&opts.ref, "ref", "", "", "Create tag using commit SHA, another tag name, or branch name (required)")
	cmd.Flags().StringVarP(&opts.msg, "message", "m", "", "the annotation of tag")
	_ = cmd.MarkFlagRequired("tag")
	_ = cmd.MarkFlagRequired("ref")
	return cmd
}

type releaseOptions struct {
	client gitlab.Client
	cfg    *config.Config
	logger *logrus.Entry

	branches []string
	tag      string
	project  string
	ref      string
	msg      string
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

	condition := func(v string) bool { return true }
	if len(o.branches) > 0 {
		condition = func(v string) bool {
			return utils.InStringArray(v, o.branches)
		}
	}

	result := releasenote.ReleaseNotesFromMR(mrs, condition)

	tagReq := gitlab.TagRequest{
		TagName:            o.tag,
		Ref:                o.ref,
		Message:            o.msg,
		ReleaseDescription: result,
	}
	if err = o.client.CreateTag(o.project, tagReq); err != nil {
		return err
	}

	fmt.Printf("successfully to release %s\n", o.tag)
	return nil
}
