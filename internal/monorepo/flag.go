package monorepo

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

type Flag []Item

func (f *Flag) String() string {
	if f == nil || len(*f) == 0 {
		return "[]"
	}

	var parts []string
	for _, item := range *f {
		parts = append(parts, fmt.Sprintf("%s:%v", item.Name, item.Paths))
	}
	return "[" + strings.Join(parts, ", ") + "]"
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
		return fmt.Errorf("parsing monorepo configuration: %w", err)
	}

	*f = Flag(items)
	return nil
}

func (f *Flag) Type() string {
	return "monorepo"
}

// GetItems returns the parsed monorepo items
func (f *Flag) GetItems() []Item {
	if f == nil {
		return nil
	}
	return []Item(*f)
}

var _ pflag.Value = (*Flag)(nil)
