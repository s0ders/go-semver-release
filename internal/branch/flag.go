package branch

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/pflag"
)

var FlagType = "branches"

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

	// Parse JSON obtained from flag value or Viper binding
	var items []Item
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return fmt.Errorf("unmarshalling %s flag value: %w", FlagType, err)
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
