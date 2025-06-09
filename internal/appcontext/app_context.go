// Package appcontext provides a structure to store the current application execution context.
//
// The use of this structure allows avoiding the use of global variables to share the states of variables across
// structures and functions.
package appcontext

import (
	"github.com/rs/zerolog"
	"github.com/spf13/viper"

	"github.com/s0ders/go-semver-release/v6/internal/branch"
	"github.com/s0ders/go-semver-release/v6/internal/monorepo"
	"github.com/s0ders/go-semver-release/v6/internal/rule"
)

type AppContext struct {
	Viper             *viper.Viper
	BranchesCfg       branch.Flag
	MonorepositoryCfg monorepo.Flag
	RulesCfg          rule.Flag
	Logger            zerolog.Logger
	CfgFile           string
	GitName           string
	GitEmail          string
	TagPrefix         string
	AccessToken       string
	RemoteName        string
	GPGKeyPath        string
	BuildMetadata     string
	DryRun            bool
	Verbose           bool
}

func New() *AppContext {
	return &AppContext{
		Viper: viper.New(),
	}
}
