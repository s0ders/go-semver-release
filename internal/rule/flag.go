package rule

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/pflag"
)

type Flag map[string][]string

const FlagType = "ruleFlag"

func (f *Flag) String() string {
	if f == nil || len(*f) == 0 {
		return "{}"
	}

	b, err := json.Marshal(f)
	if err != nil {
		return "{}"
	}

	return string(b)
}

func (f *Flag) Set(value string) error {
	var temp map[string][]string
	if err := json.Unmarshal([]byte(value), &temp); err != nil {
		return fmt.Errorf("unmarshalling rule flag value: %w", err)
	}

	*f = temp
	return nil
}

func (f *Flag) Type() string {
	return FlagType
}

var _ pflag.Value = (*Flag)(nil)
