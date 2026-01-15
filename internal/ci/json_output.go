package ci

import (
	"encoding/json"
	"fmt"
	"io"
)

// ReleaseOutput represents a single release in the JSON output.
type ReleaseOutput struct {
	NewRelease bool   `json:"new_release"`
	Version    string `json:"version"`
	Branch     string `json:"branch"`
	Project    string `json:"project,omitempty"`
	Message    string `json:"message"`
}

// Summary contains aggregate information about all releases.
type Summary struct {
	TotalCount   int `json:"total_count"`
	ReleaseCount int `json:"release_count"`
	HasReleases  bool `json:"has_releases"`
}

// JSONOutput represents the structured JSON output containing all releases and a summary.
type JSONOutput struct {
	Summary  Summary         `json:"summary"`
	Releases []ReleaseOutput `json:"releases"`
}

// NewJSONOutput creates a new JSONOutput instance.
func NewJSONOutput() *JSONOutput {
	return &JSONOutput{
		Releases: make([]ReleaseOutput, 0),
	}
}

// AddRelease adds a release to the output.
func (j *JSONOutput) AddRelease(newRelease bool, version, branch, project, message string) {
	release := ReleaseOutput{
		NewRelease: newRelease,
		Version:    version,
		Branch:     branch,
		Project:    project,
		Message:    message,
	}
	j.Releases = append(j.Releases, release)
}

// Finalize computes the summary based on added releases.
func (j *JSONOutput) Finalize() {
	j.Summary.TotalCount = len(j.Releases)
	j.Summary.ReleaseCount = 0
	j.Summary.HasReleases = false

	for _, r := range j.Releases {
		if r.NewRelease {
			j.Summary.ReleaseCount++
			j.Summary.HasReleases = true
		}
	}
}

// Write outputs the JSON to the given writer.
func (j *JSONOutput) Write(w io.Writer) error {
	j.Finalize()

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(j); err != nil {
		return fmt.Errorf("encoding JSON output: %w", err)
	}

	return nil
}
