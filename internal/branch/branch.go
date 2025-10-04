// Package branch provides functions to handle branch configuration.
package branch

import (
	"errors"
)

type Config struct {
	Items []Item `yaml:"branches" mapstructure:"branches"`
}

type Item struct {
	Name       string `yaml:"name" json:"name" mapstructure:"name"`
	Prerelease bool   `yaml:"prerelease" json:"prerelease" mapstructure:"prerelease"`
}

var (
	ErrNoBranch = errors.New("no branch configuration")
	ErrNoName   = errors.New("no name in branch configuration")
)
