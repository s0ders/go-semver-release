package monorepo

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/spf13/pflag"
)

var ErrExclusiveFlag = fmt.Errorf("the given flags are mutually exclusive")

var FlagType = "monorepo"

type Flag []Item

func (f *Flag) String() string {
	if f == nil || len(*f) == 0 {
		return "[]"
	}

	b, err := json.Marshal(f)
	if err != nil {
		return err.Error()
	}

	return string(b)
}

func (f *Flag) Set(value string) error {
	// Clear existing values
	*f = Flag{}

	if value == "" || value == "[]" {
		return nil
	}

	// Parse JSON from Viper binding
	var items []Item
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return fmt.Errorf("unmarshalling %s flag value: %w", FlagType, err)
	}

	for i, item := range items {
		if len(item.Paths) != 0 && item.Path != "" {
			return fmt.Errorf("monorepo item %q has both \"path\" and \"paths\" set: %w", item.Name, ErrExclusiveFlag)
		}

		// Normalize paths to ensure consistent matching (e.g., "./services/a/" -> "services/a")
		// Uses path.Clean (not filepath.Clean) since git always uses forward slashes
		if item.Path != "" {
			items[i].Path = path.Clean(item.Path)
		}
		for j, p := range item.Paths {
			items[i].Paths[j] = path.Clean(p)
		}
	}

	*f = items
	return nil
}

func (f *Flag) Type() string {
	return FlagType
}

// GetItems returns the parsed monorepo items
func (f *Flag) GetItems() []Item {
	if f == nil {
		return nil
	}
	return *f
}

var _ pflag.Value = (*Flag)(nil)
