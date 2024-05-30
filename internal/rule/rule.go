package rule

import "errors"

type Rules struct {
	Unmarshalled map[string][]string
}

var (
	ErrInvalidCommitType    = errors.New("invalid commit type")
	ErrInvalidReleaseType   = errors.New("invalid release type")
	ErrDuplicateReleaseRule = errors.New("duplicate release rule for the same commit type")
	ErrNoRules              = errors.New("no rule found")
)

var validCommitTypes = map[string]struct{}{
	"build":    {},
	"chore":    {},
	"ci":       {},
	"docs":     {},
	"feat":     {},
	"fix":      {},
	"perf":     {},
	"refactor": {},
	"revert":   {},
	"style":    {},
	"test":     {},
}

var validReleaseTypes = map[string]struct{}{
	"minor": {},
	"patch": {},
}

func (r Rules) Validate() error {
	if len(r.Unmarshalled) == 0 {
		return ErrNoRules
	}

	for releaseType, commitTypes := range r.Unmarshalled {
		if _, ok := validReleaseTypes[releaseType]; !ok {
			return ErrInvalidReleaseType
		}

		for _, commitType := range commitTypes {
			if _, ok := validCommitTypes[commitType]; !ok {
				return ErrInvalidCommitType
			}
		}
	}

	return nil
}

func (r Rules) Map() map[string]string {
	m := make(map[string]string)

	for releaseType, commitTypes := range r.Unmarshalled {
		for _, commitType := range commitTypes {
			m[commitType] = releaseType
		}
	}

	return m
}
