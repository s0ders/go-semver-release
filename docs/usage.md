# Usage

The program only supports a local mode of execution. This means that it requires that the repository to version is
already present on the local file system.
The program takes the path of local Git repository, computes the next SemVer and tags the repository.

This mode is a good option security-wise since it lets you use the program without having to configure any kind of
access management since it does not require any access token.

Example:

```bash
$ go-semver-release local <REPOSITORY_PATH> --rule-path <PATH> --tag-prefix <PREFIX> --release-branch <NAME> --dry-run --verbose
```

> [!TIP]
> Tag prefix can be changed during the lifetime of a repository (e.g., going from no prefix to `v`), this will
> **not** affect the SemVer tags history, the program will still be able to recognize previous SemVer tags.

For more information about available flags and their default values, run:

```bash
$ go-semver-release local --help
```

## Flags
### Tag prefix

The `--tag-prefix` flags allows to custom what will prefix the semantic version number in the Git tag. The default value
is nothing. For instance, if the next version detected is `2.0.1`, with the default tag prefix, the tag will be `2.0.1`.


A classic prefix is `v`. For instance, Go requires that your repository is versioned using the following tag format
`vX.Y.Z`.

Example:
```bash
$ go-semver-release local <PATH> --tag-prefix v
```

### Release branch

The `--release-branch` flags defines from which Git branch of your repository the commit history will be read and parsed
to compute the next semantic version number. The default value is `main`.

Example:
```bash
$ go-semver-release local <PATH> --release-branch master
```

### Release rules

Release rules define which commit type will trigger a release, and which type of release (i.e., `minor` or `patch`).

> [!IMPORTANT]
> Release type can only be `minor` or `patch`, `major` is reserved for breaking change only.

The following release rules are applied by default:
```json
{
    "rules": [
        {"type": "feat",   "release": "minor"},
        {"type": "fix",    "release": "patch"},
        {"type": "perf",   "release": "patch"},
        {"type": "revert", "release": "patch"}
    ]
}
```

You can define custom release rules to suit your needs using a JSON file and by passing it to the program as
below.

```bash
$ go-semver-release local <REPOSITORY_PATH> --rule-path ./path/to/custom/rule.json
```

If a commit type (e.g., `chore`) is not specified in you rule file, it will not trigger any kind of release.

The following `type` are supported for release rules: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`,
`revert`, `style`, `test`.

### Prerelease

A Semantic Version number can include pre-release information such as `X.Y.Z-alpha.1` or `X.Y.Z-rc`.

The `--prerelease` flag allows specifying that the generated SemVer tag, if any, is a prerelease tag and will be marked
as such by append a `rc` to the semantic version.

Example:
```bash
# Will create a tag such as "X.Y.Z-rc"
$ go-semver-release local . --prerelease
```

### Build Metadata

The Semantic Version convention states that your SemVer number can include build metadata in form
`1.2.3+<build_metadata>`. Usually, these metadata are a unique build number so that a specific version can be linked to
the build that created it.

The `--build-metadata` allows to pass a string containing build metadata that will be appended to the semantic version
number in the form stated above.

Example:
```bash
# Will produce a SemVer like "X.Y.Z+<some_job_id>"
$ go-semver-release local . --build-metadata $CI_JOB_ID
```

### GPG Signed Tags

The `--gpg-key-path` allows passing an armored GPG signing key so that the produced tags, if any, are signed with that
key.
> [!CAUTION]
> Using this flag in your CI/CD workflow means you will have to write a GPG private key to a file. Please ensure that
> this file has read and write permissions for its owner only. Furthermore, the GPG key used should be a key
> specifically generated for the purpose of signing tags. Please do not use your personal key, that way you can easily
> revoke the key if any action in your workflow came to be compromised.

Example:
```bash
$ go-semver-release local . --gpg-key-path ./path/to/key.asc
```

### Dry-run

The `--dry-run` flag controls if the repository is actually tagged after computing the next semantic version, if any.
If enabled and a new release is detected, the command JSON output will include a `next-version` key that contains the
next semantic version (without the tag prefix) that would have been applied if ran in regular mode.

Example:
```bash
$ go-semver-release local . --dry-run
```

### Verbose

The `--verbose` defines the level of verbosity that will be printed out by the command. By default, the command is not
verbose and will only print a single JSON output informing if a new was found or not, and if so, what the new (or next,
in case of dry-run mode) release is.

If enabled, the command will print whenever it finds a commit that triggers a bump in the semantic version with
information about each commit (e.g., hash, message). It will also print if it tagged the repository.

Example:
```bash
$ go-semver-release local . --verbose
```


## Command output

The `local` command output is JSON formatted so that it can be piped into tool such as `jq`.
Without the verbose flag, a single message will be printed out which keys and values depend on the scenario:

| Scenario                          | Output                                               |
| --------------------------------- | ---------------------------------------------------- |
| A new release is found            | `{"new-release": true, "new-version": "1.2.3"}`      |
| A new release is found in dry-run | `{"new-release": true, "next-version": "1.2.3"}`     |
| No new release is found           | `{"new-release": false, "current-version": "1.2.0"}` |

