package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/s0ders/go-semver-release/v6/internal/appcontext"
)

func TestCmd_Version(t *testing.T) {
	assert := assert.New(t)
	actual := new(bytes.Buffer)
	ctx := appcontext.New()

	rootCmd := NewRootCommand(ctx)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	assert.NoError(err, "local command executed with error")
}
