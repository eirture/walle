package context

import (
	"github.com/sirupsen/logrus"

	"walle/pkg/config"
	"walle/pkg/gitlab"
)

type Context struct {
	GitLabClient gitlab.Client

	Config  *config.Config
	Logger  *logrus.Entry
	Project string
}

func NewContext(client gitlab.Client, config *config.Config, logger *logrus.Entry) Context {
	return Context{
		GitLabClient: client,
		Config:       config,
		Logger:       logger,
	}
}
