package release

import (
	"fmt"

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
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&opts.branches, "branch", "b", []string{}, "the target branch name of merge request")
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "The name of a tag (required)")
	cmd.Flags().StringVarP(&opts.ref, "ref", "", "", "Create tag using commit SHA, another tag name, or branch name (required)")
	cmd.Flags().StringVarP(&opts.msg, "message", "m", "", "The annotation of tag")
	cmd.Flags().BoolVar(&opts.dry, "dry", false, "Print changelog only")
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
	dry      bool
}

func (o *releaseOptions) Run(cmd *cobra.Command, args []string) error {

	tagExists, result, err := releasenote.GetReleaseNotesByTag(
		o.client,
		o.project,
		o.tag,
		o.ref,
	)
	if err != nil {
		return err
	}

	if o.dry {
		fmt.Print(result)
		return nil
	}

	if tagExists {
		err = o.client.UpsertRelease(o.project, o.tag, result)
		if err != nil {
			return err
		}
	} else {
		tagReq := gitlab.TagRequest{
			TagName:            o.tag,
			Ref:                o.ref,
			Message:            o.msg,
			ReleaseDescription: result,
		}
		if err = o.client.CreateTag(o.project, tagReq); err != nil {
			return err
		}
	}

	fmt.Printf("successfully to release %s\n", o.tag)
	return nil
}
