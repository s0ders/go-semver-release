package helpers

const DefaultReleaseRules = `{
	"releaseRules": [
		{"type": "feat", "release": "minor"},
		{"type": "perf", "release": "minor"},
		{"type": "fix", "release": "patch"}
	]
}`