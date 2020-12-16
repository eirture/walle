package root

import (
	"os"

	"github.com/spf13/cobra"

	"walle/pkg/cmd/changelog"
	"walle/pkg/cmd/release"
	"walle/pkg/context"
)

func NewCmdRoot(ctx *context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "walle",
		Short:         "Valle is a tool which generate changelog and publish release",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	EnablePersistentFlags(ctx, cmd)
	cmd.AddCommand(release.NewReleaseCmd(ctx))
	cmd.AddCommand(changelog.NewCmdChangelog(ctx))
	return cmd
}

func EnablePersistentFlags(ctx *context.Context, cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("project", "p", "", "project fully name or id")
	cmd.PersistentFlags().String("token", "", "gitlab token")
	cmd.PersistentFlags().String("host", "", "gitlab host address")
	_ = cmd.MarkPersistentFlagRequired("project")

	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		projectOverride, _ := cmd.Flags().GetString("project")
		if projectFromEnv := os.Getenv("WALLE_PROJECT"); projectOverride == "" && projectFromEnv != "" {
			projectOverride = projectFromEnv
		}
		if projectOverride != "" {
			ctx.Project = projectOverride
		}

		tokenOverride, _ := cmd.Flags().GetString("token")
		if tokenFromEnv := os.Getenv("WALLE_GITLAB_TOKEN"); tokenFromEnv != "" && tokenOverride == "" {
			tokenOverride = tokenFromEnv
		}
		ctx.Config.Token = tokenOverride

		host, _ := cmd.Flags().GetString("host")
		if host != "" {
			ctx.Config.Host = host
		}
	}
}
