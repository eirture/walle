package version

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"walle/pkg/context"
)

func NewCmdVersion(ctx *context.Context, version, buildDate string) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "version",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(Format(version, buildDate))
		},
	}

	return cmd
}

func Format(version, buildDate string) string {
	version = strings.TrimPrefix(version, "v")

	if buildDate != "" {
		version = fmt.Sprintf("%s (%s)", version, buildDate)
	}

	return fmt.Sprintf("walle version %s\n%s\n", version, changelogURL(version))
}

func changelogURL(version string) string {
	path := "https://code.bizseer.com/liujie/walle"
	r := regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[\w.]+)?$`)
	if !r.MatchString(version) {
		return fmt.Sprintf("%s/releases", path)
	}

	url := fmt.Sprintf("%s/releases/v%s", path, strings.TrimPrefix(version, "v"))
	return url
}
