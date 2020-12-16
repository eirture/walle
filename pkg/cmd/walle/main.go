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
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
