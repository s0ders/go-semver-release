# Configuration

### How to read this page

Configuration option can be set either via flag or configuration file. The same option share the same name for its flag and configuration key counterpart. Each paragraph about an option will have an example for how to use it via flag or a YAML configuration file.

Default values are not specified here since they are available when running the following command:

```bash
$ go-semver-release release --help
```

### Configuration precedence

The order of precedence for the configuration is:

* Explicitly set flag values have the highest precedence
* Then values set in the configuration file
* Finally, flag default values have the lowest precedence, each flag default value is given in the help message of the command

### Configuration file

CLI flag: `--config`

The tool expects a configuration file for configuration options such as branches or release rules, the default path, which can be overidden, is `<REPOSITORY_ROOT>/.semver.yaml`

Example:

```bash
$ go-semver-release release <PATH> --config <CONFIG_PATH>
```

### Release rules

CLI flag: `--rules`

Release rules define which commit type will trigger a release, and which type of release (i.e., `minor` or `patch`).

{% hint style="info" %}
Release type can only be `minor` or `patch`, `major` is reserved for breaking change only which are indicated either using an exclamation mark after the commit type (e.g. `feat!`) or by stating `BREAKING CHANGE` in the commit message footer.
{% endhint %}

The following release rules are applied by default, they can be overridden by adding or removing commit types in the `minor` and `patch` list.

| Release type | Commit type             |
| ------------ | ----------------------- |
| `minor`      | `feat`                  |
| `patch`      | `fix`, `perf`, `revert` |



The following `type` are supported for release rules: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`.

Examples:

```bash
$ go-semver-release release <PATH> --rules='{"minor": ["feat"], "patch": ["fix", "perf"]}'
```

<pre class="language-yaml"><code class="lang-yaml"><strong>rules:
</strong>  minor:
    - feat
  patch:
    - fix
    - perf
    - refactor
    - revert
</code></pre>

### Branches

CLI flag: `--branches`

Branches set in configuration are the one Go Semver Release will read commit history from in order to compute the next SemVer release. In the configuration file, `branches` is a list of branch, which can have two attributes `name`, mandatory, and `prerelease` optional.

A prerelease branch will have its tag suffixed by its own name. For instance, for a branch named `rc` a set to `prerelease`, a new release will look like `1.2.3-rc`.

Examples:

```bash
$ go-semver-release release <PATH> --branches='[{"name": "master"}, {"name": "rc", "prerelease": true}]'
```

```yaml
branches:
  - name: "master"
  - name: "rc"
    prerelease: true
  - name: "alpha"
    prerelease: true
```

### Remote and access token

CLI flags: `--remote`, `--remote-name`

By default, Go Semver Release operate in local mode and expect the repository to exist on the local file system. This has the advantage of avoiding the use of access token. However, it can be easier to simply let Go Semver Release clone a repository, parse it and push the newly found SemVer tag, if any.

To enable the remote mode, you to set the following in your configuration file:

```yaml
remote: true
remote-name: "origin"
```

An access token is required so that Go Semver Release can clone the Git repository and push tags to it. All modern Git remote providers offer this feature (e.g., [GitHub](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens), [GitLab](https://docs.gitlab.com/ee/user/project/settings/project\_access\_tokens.html), [Bitbucket](https://support.atlassian.com/bitbucket-cloud/docs/access-tokens/)).

Please do not set the access token directly in the configuration file. A much safer alternative it to set the access token as a secret on the remote repository and, in your CI workflow, pass it to Go Semver Release either via the `--access-token` flag or via the `GO_SEMVER_RELEASE_ACCESS_TOKEN` environment variable.

### Monorepo

CLI flag: `--monorepo`

The program can also version separately multiple projects stored in a single repository also called "monorepo" or "mono repository". To do so, the configuration file must include a `monorepo` section stating the name and path of the various projects inside that repository.

```yaml
monorepo:
  - name: foo
    path: ./foo/
  - name: bar
    path: ./xyz/bar/
```

Each project will then be versioned separately meaning that each project will have its SemVer tag in the form `<project>-<semver>` for instance `foo-1.2.3` or `bar-v0.0.1`

**How does it work?**

The program will first fetch the latest, if any, SemVer tag for each project configured inside the `monorepo` key (e.g. `foo-1.0.0`). Then, for each project, the program will parse the commits older than the latest found tag and for each commit, will check if one of the changes made in that commit belongs to the path of that project, if so, the latest SemVer is incremented according to the type of that commit.

This means that if a commit has changes belonging to multiple projects of a monorepo, all projects concerned will have their SemVer bumped according to the commit type.

### Tag prefix

CLI flag: `--tag-prefix`

A tag prefix is used to custom the tag format of a SemVer applied to a Git repository. A classic, and the default, value is `v`. For instance, if the release version found is `1.2.3`, the Git tag will be `v1.2.3`.

{% hint style="info" %}
Tag prefix can be changed during the lifetime of a repository (e.g., going from no prefix to `v`), this will not affect the SemVer tag history, the program will still be able to recognize previous SemVer tags as long as they are annotated tags.
{% endhint %}

Example:

```bash
$ go-semver-release release <PATH> --tag-prefix v
```

### Build metadata

CLI flags: `--build-metadata`

The Semantic Version convention states that your SemVer number can include build metadata in the form `1.2.3+<build_metadata>`. Usually, these metadata represent a unique build number or build specific information so that a version can be linked to the build that created it.

The option allows to pass a string containing metadata that will be appended to the semantic version number in the form stated above.

Example:

```bash
$ go-semver-release release <PATH> --build-metadata $CI_JOB_ID
```

### GPG signed tags

CLI flag: `--gpg-key-path`

Path to an armored GPG signing key used to sign the produced tags.

{% hint style="danger" %}
Using this flag in your CI/CD workflow means you will have to write a GPG private key to a file. Please ensure that this file has read and write permissions for its owner only. Furthermore, the GPG key used should be a key specifically generated for the purpose of signing tags. Do not use your personal key, that way you can easily revoke the key if any action in your workflow came to be compromised.
{% endhint %}

{% hint style="warning" %}
As stated above, the GPG private key need to be written on disk before being read. Store it outside the repository being versioned. Because the tool first checks out to the release branch you configured, the key will disappear (since it has not been committed) and will not be found by the program.
{% endhint %}

Example:

```bash
$ go-semver-release release <PATH> --gpg-key-path ./path/to/key.asc
```

### Dry-run

CLI flag: `--dry-run`

Controls if the repository is actually tagged after computing the next semantic version.&#x20;

Example:

```bash
$ go-semver-release release <PATH> --dry-run
```

### Git name and email

CLI flags: `--git-name`, `--git-email`

The program creates new tag whenever a new release is found. These tags are annotated and, as such, require a Git signature by an author. By default, the tag will be created by an author with the name "Go Semver Release" and email "go-semver@release.ci".

Example:

```bash
$ go-semver-release release <PATH> --git-name <NAME> --git-email <EMAIL>
```

### Verbose

CLI flag: `--verbose`

Defines the level of verbosity that will be printed out by the command. By default, the command is not verbose and will only print an output informing if a new release was found along with its value.

If enabled, the command will print whenever it finds a commit that triggers a bump in the semantic version with information about each commit (e.g., hash, message) and other detailed information about the steps the program is performing.

Example:

```bash
$ go-semver-release release <PATH> --verbose
```
