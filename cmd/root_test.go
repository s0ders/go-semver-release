package cmd

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRootCmd_NoError(t *testing.T) {
	assert := assert.New(t)

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	assert.NoError(err, "should not have failed running rootCmd")
}

func TestRootCmd_Error(t *testing.T) {
	assert := assert.New(t)

	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{"unknown"})

	err := rootCmd.Execute()
	assert.Error(err, "should not have failed trying to run unknown command")
}
