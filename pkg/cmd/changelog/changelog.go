package changelog

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"walle/pkg/changelog"
	"walle/pkg/context"
	"walle/pkg/gitlab"
)

func NewCmdChangelog(ctx *context.Context) *cobra.Command {
	opts := options{
		client: ctx.GitLabClient,
		projectF: func() string {
			return ctx.Project
		},
	}

	cmd := &cobra.Command{
		Use:   "changelog",
		Short: "update changelog file by release note",
		RunE:  opts.Run,
	}

	cmd.Flags().StringVar(&opts.ref, "ref", "", "the ref name")
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "get release note from this tag (required)")
	cmd.Flags().StringVarP(&opts.branch, "branch", "b", "", "target branch name")
	cmd.Flags().StringVarP(&opts.filepath, "file", "f", "CHANGELOG.md", "the changelog file path. default is `CHANGELOG.md`")
	cmd.Flags().BoolVar(&opts.merge, "merge", false, "merge automatically")
	_ = cmd.MarkFlagRequired("tag")

	return cmd
}

type options struct {
	client   gitlab.Client
	projectF func() string
	project  string
	merge    bool

	ref      string
	branch   string
	filepath string
	tag      string
}

func (o *options) Run(cmd *cobra.Command, args []string) (err error) {
	o.project = o.projectF()
	tag, err := o.client.GetTag(o.project, o.tag)
	if err != nil {
		return
	}
	if tag.Release == nil {
		return fmt.Errorf("tag have no any release note")
	}

	fileContent, err := o.client.GetFile(o.project, o.filepath, o.ref)
	if err != nil {
		return
	}

	newContent, err := changelog.GenerateChangelog(o.tag, tag.Release.Description, fileContent, tag.Commit.CreatedAt)
	if err != nil {
		return
	}

	if newContent == fileContent {
		fmt.Println("nothing have been changed")
		return nil
	}

	msg := fmt.Sprintf("docs(changelog): update changelog of %s", o.tag)

	branchName := fmt.Sprintf("changelog-%s", o.tag)
	err = o.client.NewBranch(o.project, branchName, o.ref)
	if err == nil {
		// update file content

		fileReq := gitlab.RepoFileRequest{
			Branch:        branchName,
			CommitMessage: msg,
			Content:       newContent,
		}
		err = o.client.UpdateFile(o.project, o.filepath, fileReq)
		if err != nil {
			return
		}
	} else if !strings.Contains(err.Error(), "Branch already exists") {
		// got an error and it is not "Branch already exists"
		return
	}

	targetBranch := o.branch
	if targetBranch == "" {
		p, err := o.client.GetProject(o.project)
		if err != nil {
			return err
		}
		targetBranch = p.DefaultBranch
	}

	mrReq := gitlab.MergeRequestRequest{
		SourceBranch:       branchName,
		TargetBranch:       targetBranch,
		Title:              msg,
		Description:        msg,
		RemoveSourceBranch: true,
	}
	mr, err := o.client.CreateMergeRequest(o.project, mrReq)
	if err != nil {
		return
	}

	if o.merge {
		_, err = o.client.AcceptMR(o.project, mr.IID)
	}
	return
}
