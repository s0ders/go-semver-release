// Package monorepo provides functions to work with monorepository configuration.
package monorepo

import (
	"errors"
	"fmt"
	"path/filepath"
)

var (
	ErrNoProjects = errors.New("no projects found in configuration file despite operating in monorepo mode")
	ErrNoName     = errors.New("project has no name")
	ErrNoPath     = errors.New("project has no path")
	ErrWrongType  = errors.New("configuration value has wrong type")
)

type Project struct {
	Path      string
	Name      string
	DependsOn []string
	Release   bool
}

// Unmarshall takes a raw Viper configuration and returns a slice of Project representing various projects in a
// monorepo.
func Unmarshall(input []map[string]any) ([]Project, error) {
	if len(input) == 0 {
		return nil, ErrNoProjects
	}

	projects := make([]Project, len(input))

	for i, p := range input {

		name, ok := p["name"]
		if !ok {
			return nil, ErrNoName
		}
		var nameStr string
		if nameStr, ok = name.(string); !ok {
			return nil, ErrWrongType
		}

		path, ok := p["path"]
		if !ok {
			return nil, ErrNoPath
		}
		var pathStr string
		if pathStr, ok = path.(string); !ok {
			return nil, ErrWrongType
		}

		var dependsOn []string
		if dependsOnAny, ok := p["depends-on"]; ok {
			var dependsOnAnySlice []any
			if dependsOnAnySlice, ok = dependsOnAny.([]any); !ok {
				return nil, ErrWrongType
			}
			for _, d := range dependsOnAnySlice {
				dependsOn = append(dependsOn, fmt.Sprintf("%v", d))
			}
		}

		var release = true
		if releaseAny, ok := p["release"]; ok {
			if releaseStr, ok := releaseAny.(string); ok {
				release = releaseStr == "true" || releaseStr == "1" || releaseStr == "yes"
			} else if releaseBool, ok := releaseAny.(bool); ok {
				release = releaseBool
			} else {
				return nil, ErrWrongType
			}
		}

		project := Project{
			Name:      nameStr,
			Path:      filepath.Clean(pathStr),
			DependsOn: dependsOn,
			Release:   release,
		}

		projects[i] = project
	}

	return projects, nil
}
