# Usage

Go Semver Release requires a simple YAML configure file to work. Most of the values can be overridden either by using a flag, or environment variables. The order of precedence is as follow: 

- Explicitly set flag values have higher precedence
- Environment variable prefixed with `GO_SEMVER_RELEASE`
- Values set in the configuration file
- Flag default values



## Quickstart

All you need to get started is a Git repository, local or remote, with a commit history formatted using the [Conventional Commits](https://www.conventionalcommits.org/en/) convention and a configuration file. See example bellow for a simple configuration that will suite most use cases:

```yaml
# ~/.semver.yaml
# All parameters below are optionals and have sensible defaults except for "branches" that
# must explicitly be set.
remote: true
remote-name: "origin"
git-name: "My Custom Robot Name"
git-email: "custom-robot@acme.com"
tag-prefix: "v"
rules:
  minor:
    - feat
  patch:
    - fix
    - perf
    - revert
branches:
  - name: "master"
  - name: "alpha", prerelease: true
```

Save this configuration file at the root of you project and in your terminal or your favorite CI tool, run the following:

```bash
$ go-semver-release release <REPOSITORY_PATH_OR_URL> --config <PATH_TO_CONFIG_FILE> [--dry-run, --verbose]
```



## Configuration

As stated before, all values in the configuration files can be overridden using flags or environment variables **except** for rules and branches, these must be set explicitly in the configuration file.

To see all available flags and their default values, if any, run the following:

```bash
$ go-semver-release release --help
```



### Git name and email

Go Semver Release creates new tag whenever a new release is found. These tag are annotated and, as such, require a Git signature by an author. By default, the tag will be created by the author with the name "Go Semver Release" and the email "go-semver@release.ci".

Example:

```bash
$ go-semver-release release <PATH> --git-name <NAME> --git-email <EMAIL>
```



### Tag prefix

A tag prefix is used to custom the tag format of a SemVer applied to a Git repository. A classic, and the default, value is `v`. For instance, if the release version found is `1.2.3`, the Git tag will be `v1.2.3`. 

> [!TIP]
> Tag prefix can be changed during the lifetime of a repository (e.g., going from no prefix to `v`), this will
> **not** affect the SemVer tags history, the program will still be able to recognize previous SemVer tags as long as they are annotated tags.

Example:
```bash
$ go-semver-release local <PATH> --tag-prefix v
```



### Remote and access token

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



### Branches

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
If enabled and the command will only output if it found a new version and stop there.

Example:
```bash
$ go-semver-release release <PATH> --dry-run
```



### Verbose

The `--verbose` defines the level of verbosity that will be printed out by the command. By default, the command is not
verbose and will only print a single JSON output informing if a new release was found along with its value.

If enabled, the command will print whenever it finds a commit that triggers a bump in the semantic version with
information about each commit (e.g., hash, message) and other detailed informations about the steps the program is performing.

Example:
```bash
$ go-semver-release release <PATH> --verbose
```



## Output

The `release` command output is JSON formatted so that it can easily be parsed.

No matter the scenario (e.g., no new release, new release, dry-run) the output, if not verbose, will always have the following keys (values given for example):

```json
{
    "new-release": true,
    "version": "1.2.3",
    "branch": "master",
    "message": "new release found"
}
```

