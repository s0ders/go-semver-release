# Output

## Command output

The `release` command outputs a structured JSON object containing all releases and a summary:

```json
{
  "summary": {
    "total_count": 2,
    "release_count": 1,
    "has_releases": true
  },
  "releases": [
    {
      "new_release": true,
      "version": "1.2.3",
      "branch": "main",
      "message": "new release found"
    },
    {
      "new_release": false,
      "version": "2.0.0-rc.1",
      "branch": "rc",
      "message": "no new release"
    }
  ]
}
```

### Summary fields

| Field | Description |
|-------|-------------|
| `total_count` | Total number of branch/project combinations processed |
| `release_count` | Number of combinations that resulted in a new release |
| `has_releases` | `true` if at least one new release was created |

### Release fields

| Field | Description |
|-------|-------------|
| `new_release` | Whether this combination resulted in a new release |
| `version` | The semantic version (new or current) |
| `branch` | The branch name |
| `project` | The project name (only present in monorepo mode) |
| `message` | Status message describing the result |

## GitHub Action output

Though this tool is CI agnostic, it will try to detect if it is being executed on a GitHub Action runner.
If the program is in [monorepo](configuration.md#monorepo) mode, three outputs will be generated per branch/project pair:
* `<BRANCH_NAME>_SEMVER`, the latest semantic version
* `<BRANCH_NAME>_NEW_RELEASE`, whether a new release was found or not
* `<BRANCH_NAME>_PROJECT`, the name of the project inside the monorepo

If not in monorepo mode, two outputs will be generated per branch:
* `<BRANCH_NAME>_SEMVER`, the latest semantic version
* `<BRANCH_NAME>_NEW_RELEASE`, whether a new release was found or not
