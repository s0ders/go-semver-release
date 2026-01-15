package ci

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSONOutput(t *testing.T) {
	output := NewJSONOutput()
	assert.NotNil(t, output)
	assert.Empty(t, output.Releases)
	assert.Equal(t, 0, output.Summary.TotalCount)
	assert.Equal(t, 0, output.Summary.ReleaseCount)
	assert.False(t, output.Summary.HasReleases)
}

func TestJSONOutput_AddRelease(t *testing.T) {
	output := NewJSONOutput()

	output.AddRelease(true, "1.0.0", "main", "", "new release found")
	assert.Len(t, output.Releases, 1)
	assert.Equal(t, "1.0.0", output.Releases[0].Version)
	assert.Equal(t, "main", output.Releases[0].Branch)
	assert.True(t, output.Releases[0].NewRelease)
	assert.Empty(t, output.Releases[0].Project)

	output.AddRelease(false, "2.0.0-rc.1", "rc", "api", "no new release")
	assert.Len(t, output.Releases, 2)
	assert.Equal(t, "2.0.0-rc.1", output.Releases[1].Version)
	assert.Equal(t, "rc", output.Releases[1].Branch)
	assert.False(t, output.Releases[1].NewRelease)
	assert.Equal(t, "api", output.Releases[1].Project)
}

func TestJSONOutput_Finalize(t *testing.T) {
	tests := []struct {
		name           string
		releases       []ReleaseOutput
		wantTotal      int
		wantReleases   int
		wantHasRelease bool
	}{
		{
			name:           "no releases",
			releases:       []ReleaseOutput{},
			wantTotal:      0,
			wantReleases:   0,
			wantHasRelease: false,
		},
		{
			name: "all new releases",
			releases: []ReleaseOutput{
				{NewRelease: true, Version: "1.0.0", Branch: "main"},
				{NewRelease: true, Version: "2.0.0", Branch: "develop"},
			},
			wantTotal:      2,
			wantReleases:   2,
			wantHasRelease: true,
		},
		{
			name: "no new releases",
			releases: []ReleaseOutput{
				{NewRelease: false, Version: "1.0.0", Branch: "main"},
				{NewRelease: false, Version: "2.0.0", Branch: "develop"},
			},
			wantTotal:      2,
			wantReleases:   0,
			wantHasRelease: false,
		},
		{
			name: "mixed releases",
			releases: []ReleaseOutput{
				{NewRelease: true, Version: "1.0.0", Branch: "main"},
				{NewRelease: false, Version: "2.0.0-rc.1", Branch: "rc"},
				{NewRelease: true, Version: "1.1.0", Branch: "main", Project: "api"},
			},
			wantTotal:      3,
			wantReleases:   2,
			wantHasRelease: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &JSONOutput{Releases: tt.releases}
			output.Finalize()

			assert.Equal(t, tt.wantTotal, output.Summary.TotalCount)
			assert.Equal(t, tt.wantReleases, output.Summary.ReleaseCount)
			assert.Equal(t, tt.wantHasRelease, output.Summary.HasReleases)
		})
	}
}

func TestJSONOutput_Write(t *testing.T) {
	output := NewJSONOutput()
	output.AddRelease(true, "1.0.0", "main", "", "new release found")
	output.AddRelease(false, "2.0.0-rc.1", "rc", "api", "no new release")

	var buf bytes.Buffer
	err := output.Write(&buf)
	require.NoError(t, err)

	// Verify JSON is valid and contains expected data
	var result JSONOutput
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, 2, result.Summary.TotalCount)
	assert.Equal(t, 1, result.Summary.ReleaseCount)
	assert.True(t, result.Summary.HasReleases)
	assert.Len(t, result.Releases, 2)

	// Verify first release
	assert.True(t, result.Releases[0].NewRelease)
	assert.Equal(t, "1.0.0", result.Releases[0].Version)
	assert.Equal(t, "main", result.Releases[0].Branch)
	assert.Empty(t, result.Releases[0].Project)

	// Verify second release
	assert.False(t, result.Releases[1].NewRelease)
	assert.Equal(t, "2.0.0-rc.1", result.Releases[1].Version)
	assert.Equal(t, "rc", result.Releases[1].Branch)
	assert.Equal(t, "api", result.Releases[1].Project)
}

func TestJSONOutput_Write_EmptyReleases(t *testing.T) {
	output := NewJSONOutput()

	var buf bytes.Buffer
	err := output.Write(&buf)
	require.NoError(t, err)

	var result JSONOutput
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, 0, result.Summary.TotalCount)
	assert.Equal(t, 0, result.Summary.ReleaseCount)
	assert.False(t, result.Summary.HasReleases)
	assert.Empty(t, result.Releases)
}

func TestJSONOutput_ProjectOmittedWhenEmpty(t *testing.T) {
	output := NewJSONOutput()
	output.AddRelease(true, "1.0.0", "main", "", "new release found")

	var buf bytes.Buffer
	err := output.Write(&buf)
	require.NoError(t, err)

	// Verify that "project" key is omitted when empty
	jsonStr := buf.String()
	assert.NotContains(t, jsonStr, `"project"`)
}

func TestJSONOutput_ProjectIncludedWhenSet(t *testing.T) {
	output := NewJSONOutput()
	output.AddRelease(true, "1.0.0", "main", "api", "new release found")

	var buf bytes.Buffer
	err := output.Write(&buf)
	require.NoError(t, err)

	// Verify that "project" key is included when set
	jsonStr := buf.String()
	assert.Contains(t, jsonStr, `"project": "api"`)
}
