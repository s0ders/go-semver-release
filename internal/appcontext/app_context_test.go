package appcontext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	ctx := New()

	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.Viper)
}

func TestAppContext_DefaultValues(t *testing.T) {
	ctx := New()

	// Check default values are zero/empty
	assert.Empty(t, ctx.CfgFile)
	assert.Empty(t, ctx.GitName)
	assert.Empty(t, ctx.GitEmail)
	assert.Empty(t, ctx.TagPrefix)
	assert.Empty(t, ctx.AccessToken)
	assert.Empty(t, ctx.RemoteName)
	assert.Empty(t, ctx.GPGKeyPath)
	assert.Empty(t, ctx.BuildMetadata)
	assert.False(t, ctx.DryRun)
	assert.False(t, ctx.Verbose)
	assert.False(t, ctx.LightweightTags)
}
