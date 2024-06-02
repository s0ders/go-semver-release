package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type Branches []map[string]string

// TODO: refactor, variabilize
func TestRoot_ConfigParsing(t *testing.T) {
	assert := assert.New(t)

	tmpConfigDir, err := initViperConfig()
	if err != nil {
		t.Fatalf("initializing viper config: %s", err)
	}

	defer func() {
		err = os.RemoveAll(tmpConfigDir)
		if err != nil {
			t.Fatalf("removing temp config dir: %s", err)
		}
	}()

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"--git-name", "Override Git Name", "--help"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("executing root command: %s", err)
	}

	assert.Equal("v", viper.GetString("tag-prefix"), "tag prefix should match")
	assert.Equal("Override Git Name", viper.GetString("git-name"), "git name should match")
	assert.Equal("ci-robot@git.sh", viper.GetString("git-email"), "git email should match")

	wantRules := map[string][]string{"minor": {"feat"}, "patch": {"fix", "perf", "revert"}}
	assert.Equal(wantRules, viper.GetStringMapStringSlice("rules"), "rules should match")

	wantBranches := Branches{{"pattern": "main"}, {"pattern": "dev/*", "prerelease": "1", "prerelease-suffix": "rc", "prerelease-incremental": "1"}}
	gotBranches := Branches{}

	err = viper.UnmarshalKey("branches", &gotBranches)
	if err != nil {
		t.Fatalf("unmarshaling branches key: %s", err)
	}

	assert.Equal(wantBranches, gotBranches, "branches should match")
}

func initViperConfig() (path string, err error) {
	config := `
{
  "git-name": "My Robot",
  "git-email": "ci-robot@git.sh",
  "tag-prefix": "v",
  "branches": [
    {"pattern": "main"},
    {"pattern": "dev/*", "prerelease": true, "prerelease-suffix": "rc", "prerelease-incremental": true}
  ],
  "rules": {
    "minor": ["feat"],
    "patch": ["fix", "perf", "revert"]
  }
}
`

	dir, err := os.MkdirTemp("", "config-*")

	defer func() {
		err = os.RemoveAll(dir)
	}()

	filePath := filepath.Join(dir, ".semver.json")

	err = os.WriteFile(filePath, []byte(config), 0666)
	if err != nil {
		return "", err
	}

	viper.SetConfigFile(filePath)

	if err = viper.ReadInConfig(); err == nil {
		return "", err
	}

	return dir, err
}
