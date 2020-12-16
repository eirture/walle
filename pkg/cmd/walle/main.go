package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"walle/pkg/cmd/root"
	"walle/pkg/config"
	"walle/pkg/context"
	"walle/pkg/gitlab"
)

func main() {

	cfg := config.LoadConfig()

	if hostFromEnv := os.Getenv("WALLE_GITLAB_HOST"); hostFromEnv != "" {
		cfg.Host = hostFromEnv
	}

	logger := logrus.WithField("client", "walle")
	client := gitlab.NewClient(logger, &cfg)
	ctx := context.NewContext(client, &cfg, logger)
	rootCmd := root.NewCmdRoot(&ctx)
	var expandedArgs []string
	if len(os.Args) > 0 {
		expandedArgs = os.Args[1:]
	}

	rootCmd.SetArgs(expandedArgs)
	err := rootCmd.Execute()
	if err != nil {
		er(err)
	}

	//c := gitlab.NewClient(logrus.Fields{}, func() []byte {
	//	return []byte("YmLCdp63yPqqFfsdppCH")
	//}, "https://code.bizseer.com/api/v4/")
	//t, err := time.ParseInLocation(time.RFC3339, "2020-12-10T09:41:58.975Z", time.Local)
	//if err != nil {
	//	er(err)
	//}
	//
	//project := "liujie/gitlab-ci-sample-java"
	//
	//mrs, err := c.ListMergeRequests(project, t)
	////tags, err := c.ListTags("efficiency/ticket-backend")
	////if err != nil {
	////	er(err)
	////}
	////for _, tag := range tags {
	////	fmt.Println(tag.Name, tag.Target)
	////}
	//var releases []string
	//for _, mr := range mrs {
	//	releases = append(releases, mr.Title)
	//	fmt.Println(mr.Title, mr.State, mr.MergedAt, mr.Labels)
	//}
	//
	//err = c.CreateTag(project, gitlab.TagRequest{
	//	TagName:            "v1.0.1",
	//	Ref:                "f24d7d3d38eea95b295e7abd2e8dd8323bb2e0b9",
	//	Message:            "test create tag",
	//	ReleaseDescription: fmt.Sprintf("some release message:\n\n- %s", strings.Join(releases, "\n- ")),
	//})
	//
	//if err != nil {
	//	er(err)
	//}
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}
