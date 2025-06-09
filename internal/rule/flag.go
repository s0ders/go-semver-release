package rule

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/pflag"
)

type Flag map[string]string

const FlagType = "rules"

func (f *Flag) String() string {
	if f == nil || len(*f) == 0 {
		return "{}"
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

	var rules map[string][]string

	if err := json.Unmarshal([]byte(value), &rules); err != nil {
		return fmt.Errorf("unmarshalling %s flag value: %w", FlagType, err)
	}

	commitTypesHashmap := make(map[string]string)

	for releaseType, commitTypes := range rules {
		if _, ok := ValidReleaseTypes[releaseType]; !ok {
			return fmt.Errorf("%w: %q", ErrInvalidReleaseType, releaseType)
		}

		for _, commitType := range commitTypes {
			if _, ok := ValidCommitTypes[commitType]; !ok {
				return fmt.Errorf("%w: %q", ErrInvalidCommitType, commitType)
			}

			if _, ok := commitTypesHashmap[commitType]; ok {
				return fmt.Errorf("%w: %q", ErrDuplicateReleaseRule, commitType)
			}

			commitTypesHashmap[commitType] = releaseType
		}
	}

	*f = commitTypesHashmap

	return nil
}

func (f *Flag) Type() string {
	return FlagType
}

var _ pflag.Value = (*Flag)(nil)
