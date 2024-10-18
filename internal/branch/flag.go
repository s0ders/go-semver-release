package branch

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/pflag"
)

type Flag []map[string]any

const FlagType = "JSON string"

func (f *Flag) String() string {
	if f == nil || len(*f) == 0 {
		return "[]"
	}

	b, err := json.Marshal(f)
	if err != nil {
		return "[]"
	}

	return string(b)
}

func (f *Flag) Set(value string) error {
	var temp []map[string]any

	if err := json.Unmarshal([]byte(value), &temp); err != nil {
		return fmt.Errorf("unmarshalling branch flag value: %w", err)
	}

	*f = temp
	return nil
}

func (f *Flag) Type() string {
	return FlagType
}

var _ pflag.Value = (*Flag)(nil)
