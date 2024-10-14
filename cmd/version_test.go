package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmd_Version(t *testing.T) {
	assert := assert.New(t)
	actual := new(bytes.Buffer)
	ctx := NewAppContext()

	rootCmd := NewRootCommand(ctx)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	assert.NoError(err, "local command executed with error")
}
