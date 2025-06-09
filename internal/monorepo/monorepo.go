// Package monorepo provides functions to work with monorepository configuration.
package monorepo

type Project struct {
	Path string
	Name string
}

// TOD0: where to tell viper about this?
type Config struct {
	Items []Item `yaml:"monorepo" mapstructure:"monorepo"`
}

type Item struct {
	Name  string   `yaml:"name" mapstructure:"name"`
	Paths []string `yaml:"paths" mapstructure:"paths"`
}
