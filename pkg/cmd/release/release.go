package release

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"walle/pkg/changelog"
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
	cmd.Flags().StringVar(&opts.changelog, "changelog", "", "the changelog file path")
	cmd.Flags().StringVar(&opts.changelogBranch, "changelog-branch", "", "the target branch name of changelog MR")
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

	changelog       string
	changelogBranch string
}

func (o *releaseOptions) Run(cmd *cobra.Command, args []string) error {

	tagExists, result, err := releasenote.ChangelogFromMR(
		o.client,
		o.project,
		o.tag,
		o.branches,
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

	if o.changelog != "" {
		var mr *gitlab.MergeRequest
		if mr, err = o.createChangelogMR(o.tag, result, o.changelog); err != nil {
			return err
		}
		fmt.Printf("A merge request `%s` has been created\n", mr.Title)
	}

	return nil
}

func (o *releaseOptions) createChangelogMR(tag, content, filepath string) (mr *gitlab.MergeRequest, err error) {
	newContent, err := changelog.GenerateChangelog(tag, content, filepath)
	if err != nil {
		return
	}
	branchName := fmt.Sprintf("changelog-%s", tag)
	targetBranch := o.changelogBranch
	if targetBranch == "" {
		p, err := o.client.GetProject(o.project)
		if err != nil {
			return nil, err
		}
		targetBranch = p.DefaultBranch
	}
	msg := fmt.Sprintf("docs(changelog): update changelog of %s", tag)
	ufReq := gitlab.RepoFileRequest{
		Branch:        branchName,
		CommitMessage: msg,
		Content:       newContent,
	}
	if err = o.client.UpdateFile(o.project, filepath, ufReq); err != nil {
		return
	}

	mrReq := gitlab.MergeRequestRequest{
		SourceBranch:       branchName,
		TargetBranch:       targetBranch,
		Title:              msg,
		Description:        fmt.Sprintf("Update changelog of version %s by [walle](https://code.bizseer.com/liujie/walle)", tag),
		RemoveSourceBranch: true,
	}
	mr, err = o.client.CreateMergeRequest(o.project, mrReq)
	if err != nil {
		return
	}

	mr, err = o.client.AcceptMR(o.project, mr.IID)
	return
}
