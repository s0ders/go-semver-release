// Package monorepo provides functions to work with monorepository configuration.
package monorepo

type Config struct {
	Items []Item `yaml:"monorepo" json:"monorepo" mapstructure:"monorepo"`
}

type Item struct {
	Name  string   `yaml:"name" json:"name" mapstructure:"name"`
	Path  string   `yaml:"path" json:"path" mapstructure:"path"`
	Paths []string `yaml:"paths" json:"paths" mapstructure:"paths"`
}
