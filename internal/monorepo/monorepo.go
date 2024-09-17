package monorepo

import "errors"

var (
	ErrNoProjects = errors.New("no projects found in configuration file despite operating in monorepo mode")
	ErrNoName     = errors.New("project has no name")
	ErrNoPath     = errors.New("project has no path")
)

type Project struct {
	Path string
	Name string
}

// Unmarshall takes a raw Viper configuration and returns a slice of Project representing various projects in a
// monorepo.
func Unmarshall(input []map[string]string) ([]Project, error) {
	if len(input) == 0 {
		return nil, ErrNoProjects
	}

	projects := make([]Project, len(input))

	for i, p := range input {

		name, ok := p["name"]
		if !ok {
			return nil, ErrNoName
		}

		path, ok := p["path"]
		if !ok {
			return nil, ErrNoPath
		}

		project := Project{
			Name: name,
			Path: path,
		}

		projects[i] = project
	}

	return projects, nil
}
