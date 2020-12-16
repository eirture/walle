package config

import "fmt"

var (
	defaultHost = "http://gitlab.com"
)

type Config struct {
	Host  string
	Token string
}

func (c *Config) GetAPIBase() string {
	return fmt.Sprintf("%s/api/v4", c.Host)
}

func (c *Config) GetToken() string {
	return c.Token
}
func LoadConfig() Config {
	return Config{
		Host: defaultHost,
	}
}
