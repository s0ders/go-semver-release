# Usage



## Quickstart

## Configuration

As stated before, all values in the configuration files can be overridden using flags or environment variables **except** for rules and branches, these must be set explicitly in the configuration file.

To see all available flags and their default values, if any, run the following:

```bash
$ go-semver-release release --help
```

All available flags can be set using environment variable by capitalizing them, using snake case and prefixing them with `GO_SEMVER_RELEASE`. For instance, you can set the tag prefix as below:

```
$ GO_SEMVER_RELEASE_TAG_PREFIX="v"
```

## Output



> \[!IMPORTANT]&#x20;
