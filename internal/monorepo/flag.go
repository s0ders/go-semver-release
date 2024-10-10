package monorepo

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/pflag"
)

type Flag []map[string]string

func (f *Flag) String() string {
	if f == nil {
		return "[]"
	}

	b, err := json.Marshal(f)
	if err != nil {
		return "[]"
	}

	return string(b)
}

func (f *Flag) Set(value string) error {
	var temp []map[string]string
	if err := json.Unmarshal([]byte(value), &temp); err != nil {
		return fmt.Errorf("unmarshalling monorepo flag value: %w", err)
	}

	*f = temp
	return nil
}

func (f *Flag) Type() string {
	return "monorepoFlag"
}

func (f *Flag) Projects() ([]Project, error) {
	m := []map[string]string(*f)

	projects, err := Unmarshall(m)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling projects: %w", err)
	}

	return projects, nil
}

var _ pflag.Value = (*Flag)(nil)
