// Package appcontext provides a structure to store the current application execution context.
//
// The use of this structure allows to avoid the use of global variables to share the states of variables across
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
	Viper              *viper.Viper
	Branches           []branch.Branch
	Projects           []monorepo.Project
	Rules              rule.Rules
	BranchesFlag       branch.Flag
	MonorepositoryFlag monorepo.Flag
	RulesFlag          rule.Flag
	Logger             zerolog.Logger
	CfgFileFlag        string
	GitNameFlag        string
	GitEmailFlag       string
	TagPrefixFlag      string
	AccessTokenFlag    string
	RemoteNameFlag     string
	GPGKeyPathFlag     string
	BuildMetadataFlag  string
	DryRunFlag         bool
	VerboseFlag        bool
}

func New() *AppContext {
	return &AppContext{
		Viper: viper.New(),
	}
}
