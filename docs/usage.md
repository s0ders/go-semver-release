# Usage

Go Semver Release requires a simple YAML configure file to work. Most of the values can be overridden either by using a 
flag, or environment variables. The order of precedence is as follows: 

- Explicitly set flag values have the highest precedence
- Then environment variable prefixed with `GO_SEMVER_RELEASE`
- Then values set in the configuration file
- Finally, flag default values have the lowest precedence



## Quickstart

All you need to get started is a Git repository, local or remote, with a commit history formatted using the 
[Conventional Commits](https://www.conventionalcommits.org/en/) convention and a configuration file. The following example configuration file suites 
most use cases:

```yaml
# ~/.semver.yaml
# All parameters below are optionals and have sensible defaults except for "branches" that must explicitly be set.
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
  - name: "alpha"
    prerelease: true
```

Save this configuration file at the root of you project and in your terminal or your favorite CI tool, run the 
following:

```bash
$ go-semver-release release <REPOSITORY_PATH_OR_URL> --config <PATH_TO_CONFIG_FILE> [--dry-run, --verbose]
```

## Configuration

As stated before, all values in the configuration files can be overridden using flags or environment variables 
**except** for rules and branches, these must be set explicitly in the configuration file.

To see all available flags and their default values, if any, run the following:

```bash
$ go-semver-release release --help
```

All available flags can be set using environment variable by capitalizing them, using snake case and prefixing them 
with `GO_SEMVER_RELEASE`. For instance, you can set the tag prefix as below:

```
$ GO_SEMVER_RELEASE_TAG_PREFIX="v"
```

### Branches

Branches, in the configuration, define which Git branches are release branches. Release branches are the one Go Semver 
Release will read commit history from in order to compute the next SemVer release. In the configuration file, 
`branches` is a list of branch, which can have two attributes `name`, mandatory, and `prerelease` optional. 

A prerelease branch will have its tag suffixed by its own name. For instance, for a branch named `rc` a set to 
`prerelease`, a new release will look like `v1.2.3-rc`.

Example:

```yaml
# ~/.semver.yaml
branches:
  - name: "master"
  - name: "rc"
    prerelease: true
  - name: "alpha"
    prerelease: true
# ...
```

### Remote and access token

By default, Go Semver Release operate in local mode and expect the repository to exist on the local file system. This
has the advantage of avoiding the use of access token. However, it can be easier to simply let Go Semver Release clone
a repository, parse it and push the newly found SemVer tag, if any.

To enable the remote mode, you to set the following in your configuration file:

```yaml
# ~/.semver.yaml
remote: true
remote-name: "origin"
# ...
```

You also need an access token so that Go Semver Release can clone your repository and push tags to it. All modern Git
remote providers offer this feature (e.g., [GitHub](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens), [GitLab](https://docs.gitlab.com/ee/user/project/settings/project_access_tokens.html), [Bitbucket](https://support.atlassian.com/bitbucket-cloud/docs/access-tokens/)).

Please do not set your access token directly in your configuration file. A much safer alternative it to set the access
token as a secret on your repository and, in your CI workflow, pass it to Go Semver Release either via the
`--access-token` flag or via the `GO_SEMVER_RELEASE_ACCESS_TOKEN` environment variable.


### Release rules

Release rules define which commit type will trigger a release, and which type of release (i.e., `minor` or `patch`).

> [!IMPORTANT]
> Release type can only be `minor` or `patch`, `major` is reserved for breaking change only.

The following release rules are applied by default, they can be overridden by adding or removing commit types in the
`minor` and `patch` list.
```yaml
rules:
  minor:
    - feat
  patch:
    - fix
    - perf
    - revert
```

The following `type` are supported for release rules: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`,
`revert`, `style`, `test`.

### Tag prefix

A tag prefix is used to custom the tag format of a SemVer applied to a Git repository. A classic, and the default, value
is `v`. For instance, if the release version found is `1.2.3`, the Git tag will be `v1.2.3`.

> [!TIP]
> Tag prefix can be changed during the lifetime of a repository (e.g., going from no prefix to `v`), this will **not**
> affect the SemVer tags history, the program will still be able to recognize previous SemVer tags as long as they are
> annotated tags.

Example:
```bash
$ go-semver-release release <PATH> --tag-prefix v
```

### Multiple projects in a single repository or "monorepo"

The program can also version separately multiple projects stored in a single repository also called "monorepo" or "mono 
repository". To do so, the configuration file must include a `monorepo` section stating the name and path of the various
projects inside that repository.

```yaml
monorepo:
  - name: foo
    path: ./foo/
  - name: bar
    path: ./xyz/bar/
```

Each project will then be versioned separately meaning that each project will have its SemVer tag in the form 
`<project>-<semver>` for instance `foo-1.2.3` or `bar-v0.0.1`

**How does it work?**

The program will first fetch the latest, if any, SemVer tag for each project configured inside the `monorepo` key 
(e.g. `foo-1.0.0`). Then, for each project, the program will parse the commits older than the latest found tag and for 
each commit, will check if one of the changes made in that commit belongs to the path of that project, if so, the latest
SemVer is incremented according to the type of that commit.

This means that if a commit has changes belonging to multiple projects of a monorepo, all projects concerned will have 
their SemVer bumped according to the commit type.

### Build metadata

The Semantic Version convention states that your SemVer number can include build metadata in form
`1.2.3+<build_metadata>`. Usually, these metadata represent a unique build number or a build specific information so 
that a specific version can be linked to the build that created it.

The `--build-metadata` allows to pass a string containing build metadata that will be appended to the semantic version
number in the form stated above.

Example:
```bash
# Will produce a SemVer like "X.Y.Z+<some_job_id>"
$ go-semver-release release <PATH> --build-metadata $CI_JOB_ID
```

### GPG signed tags

The `--gpg-key-path` allows passing an armored GPG signing key so that the produced tags, if any, are signed with that
key.
> [!CAUTION]
> Using this flag in your CI/CD workflow means you will have to write a GPG private key to a file. Please ensure that 
> this file has read and write permissions for its owner only. Furthermore, the GPG key used should be a key 
> specifically generated for the purpose of signing tags. Please do not use your personal key, that way you can easily 
> revoke the key if any action in your workflow came to be compromised.

Example:
```bash
$ go-semver-release release <PATH> --gpg-key-path ./path/to/key.asc
```

### Dry-run

The `--dry-run` flag controls if the repository is actually tagged after computing the next semantic version, if any.
If enabled and the command will only output if it found a new version and stop there.

Example:
```bash
$ go-semver-release release <PATH> --dry-run
```

### Git name and email

Go Semver Release creates new tag whenever a new release is found. These tag are annotated and, as such, require a Git
signature by an author. By default, the tag will be created by the author with the name "Go Semver Release" and email
"go-semver@release.ci".

Example:

```bash
$ go-semver-release release <PATH> --git-name <NAME> --git-email <EMAIL>
```

### Verbose

The `--verbose` defines the level of verbosity that will be printed out by the command. By default, the command is not
verbose and will only print a single JSON output informing if a new release was found along with its value.

If enabled, the command will print whenever it finds a commit that triggers a bump in the semantic version with
information about each commit (e.g., hash, message) and other detailed information about the steps the program is 
performing.

Example:
```bash
$ go-semver-release release <PATH> --verbose
```

## Output

The `release` command output is JSON formatted so that it can easily be parsed.

If executed in non-verbose mode, no matter the scenario (e.g., no new release, new release, dry-run) the output, will 
always have the following keys (values are given for example), and the program will produce one of these output per 
branch parsed:

```json
{
    "new-release": true,
    "version": "1.2.3",
    "branch": "master",
    "project": "foo",
    "message": "new release found"
}
```

> [!IMPORTANT]
> The `project` key will only be present in an output if executed in monorepo mode. See the "Multiple projects in a 
> single repository or "monorepo"" section for more information.
