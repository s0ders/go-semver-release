// Package branch provides functions to handle branch configuration.
package branch

type Config struct {
	Items []Item `yaml:"branches" mapstructure:"branches"`
}

type Item struct {
	Name       string `yaml:"name" json:"name" mapstructure:"name"`
	Prerelease bool   `yaml:"prerelease" json:"prerelease" mapstructure:"prerelease"`
}
