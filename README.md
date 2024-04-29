![GitHub Tag](https://img.shields.io/github/v/tag/s0ders/go-semver-release?label=Version&color=bb33ff) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/s0ders/go-semver-release)
[![Go Reference](https://pkg.go.dev/badge/github.com/s0ders/go-semver-release.svg)](https://pkg.go.dev/github.com/s0ders/go-semver-release/v2)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/s0ders/go-semver-release/main.yaml?label=CI&color=17dd42)
[![Go Report Card](https://goreportcard.com/badge/github.com/s0ders/go-semver-release/v2)](https://goreportcard.com/report/github.com/s0ders/go-semver-release/v2) ![Codecov](https://img.shields.io/codecov/c/github/s0ders/go-semver-release?label=Coverage) ![GitHub License](https://img.shields.io/github/license/s0ders/go-semver-release?label=License)



# Go Semver Release

Go program designed to automate versioning of Git repository by analyzing their formatted commit history and tagging them with the right semver number. This program can be used directly via its CLI or its corresponding [GitHub Action](https://github.com/marketplace/actions/go-semver-release).

<ul>
    <li><a href="#Motivation">Motivation</a></li>
    <li><a href="#Install">Install</a></li>
    <li><a href="#Usage">Usage</a></li>
    <li><a href="#github-actions">GitHub Actions</a></li>
    <li><a href="#release-rules">Release Rules</a></li>
</ul>


## Motivation

This project was built to create a lightweight and simple tool to seamlessly automate versioning on your Git repository. Following the UNIX philosophy of "make each program do one thing well", it only handles publishing semver tags to your Git repository, no package publishing or any other features. 

The Docker image merely weight 7MB and the Go program inside will compute your [semver](https://semver.org) tag in seconds, no matter the size of your commit history.

To use this tool, all you have to do is:

- Choose a release branch (e.g., `main`)

- Take care to format commits on that branch by following the [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) convention, which many IDEs plugins offers to do seamlessly (e.g., [VSCode](https://marketplace.visualstudio.com/items?itemName=vivaxy.vscode-conventional-commits), [IntelliJ](https://plugins.jetbrains.com/plugin/13389-conventional-commit))

> [!IMPORTANT]
> `go-semver-release` can only read **annotated** Git tags, so if you plan on only using it in dry-run mode to then use its output to tag your repository with an other action, make sure the tag you are pushing is annotated, otherwise the program will not be able to detect it during its next execution.

## Install

If [Go](https://go.dev) is installed on your machine, you can install from source using `go install`:

```bash
$ go install github.com/s0ders/go-semver-release@latest
$ go-semver-release --help
```

For cross-platform compatibility, you can use the generated [Docker image](https://hub.docker.com/r/soders/go-semver-release/tags):

```bash
$ docker pull soders/go-semver-release:latest
$ docker run --rm soders/go-semver-release --help
```

For security purposes, each Docker image comes with a corresponding SBOM.
## Prerequisites

- The commits of the Git repository to version must follow the [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) convention.
- The Git repository must already be initialized (i.e., Git `HEAD` does not point to nothing)

## Usage

The CLI supports two mode of execution: remote and local.

In remote mode, the program will attempt to clone the remote repository using the provided URL and access token to then compute the next semver, tag it onto the cloned repository and push it back to the remote.

In local mode, the program takes the path of the already present Git repository, computes the next semver, tags the local repository with it and stops. This mode is a good option security-wise since it lets you use the program without having to configure any kind of right management because it does not require any access token.

Remote mode example:

```bash
$ go-semver-release remote --git-url <URL> --rules-path <PATH> --token <TOKEN> \
                           --tag-prefix <PREFIX> --release-branch <NAME> --dry-run --verbose
```

Local mode example:

```bash
$ go-semver-release local <REPOSITORY_PATH> --rules-path <PATH> --tag-prefix <PREFIX> \
                                            --release-branch <NAME> --dry-run --verbose
```

> [!TIP]
> You can change your tag prefix during the lifetime of your repository (e.g., going from none to `v`) and this will **not** affect your semver tags history, meaning that the program will still be able to recognize semver tags made with your old-prefixes, if any. There are no limitation to how many time you can change your tag prefix during the lifetime of your repository.

For more informations about commands and flags usage as well as the default value, simply run:

```bash
$ go-semver-release <COMMAND> --help
```





## GitHub Actions

### Inputs

The action takes the same parameters as those defined in the <a href="#Usage">usage</a> section. Note that the boolean flags (e.g., `--dry-run`, `--verbose`) need to be passed as a string inside your YAML work-flow due to how Github Actions works.

### Outputs

The action generate two outputs 
- `SEMVER`, the computed semver or the current one if no new were computed, prefixed with the given `tag-prefix` if any;
- `NEW_RELEASE`, whether a new semver was computed or not.

## Release Rules

Release rules define which commit type will trigger a release, and what type of release (i.e. `major`, `minor`, `patch`). **By default**, the program applies the following release rules:

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

You can define custom release rules to suit your needs using a JSON or YAML file and by passing it to the program as bellow. Be careful with release rules though, especially major ones, as their misuse might easily make you loose the benefits of using a semantic version number.

```yaml
rules:
  - type: feat
    release: minor
  - type: fix
    release: patch
  - type: perf
    release: patch
  - type: refactor
    release: patch
```

If a commit type (e.g., `chore`) is not specified in you rule file, it won't trigger any kind of release.

The following `type` are supported for release rules: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`.

The following `release` types are supported for release rules: `major`, `minor`, `patch`.

## Work in progress
- [ ] Support non-annotated tags
- [ ] Fix local action (Docker volumes)
- [ ] Make "remote" command a flag
- [ ] Create /docs/ folder for clarity
- [ ] Create version cmd with build info (version, commit hash, builder number)
- [ ] Centralize logging