package rule

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/pflag"
)

type Flag map[string][]string

func (f *Flag) String() string {
	if f == nil {
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
	return "ruleFlag"
}

func (f *Flag) Rules() (Rules, error) {
	m := map[string][]string(*f)

	rules, err := Unmarshall(m)
	if err != nil {
		return Rules{}, fmt.Errorf("unmarshalling rules: %w", err)
	}

	return rules, nil
}

var _ pflag.Value = (*Flag)(nil)
