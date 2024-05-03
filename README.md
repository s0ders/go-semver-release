![GitHub Tag](https://img.shields.io/github/v/tag/s0ders/go-semver-release?label=Version&color=bb33ff) 
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/s0ders/go-semver-release)
[![Go Reference](https://pkg.go.dev/badge/github.com/s0ders/go-semver-release.svg)](https://pkg.go.dev/github.com/s0ders/go-semver-release/v2)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/s0ders/go-semver-release/main.yaml?label=CI)
[![Go Report Card](https://goreportcard.com/badge/github.com/s0ders/go-semver-release/v2)](https://goreportcard.com/report/github.com/s0ders/go-semver-release/v2) 
![Codecov](https://img.shields.io/codecov/c/github/s0ders/go-semver-release?label=Coverage) 
![GitHub License](https://img.shields.io/github/license/s0ders/go-semver-release?label=License)

# Go Semver Release

Go program designed to automate versioning of Git repository by analyzing their formatted commit history and tagging 
them with the right semver number. 

<ul>
    <li><a href="#Motivation">Motivation</a></li>
    <li><a href="#Install">Install</a></li>
    <li><a href="#Usage">Usage</a></li>
    <li><a href="#ci-workflow-examples">CI workflow examples</a></li>
</ul>

## Motivation

This project was built to create a lightweight and simple tool to seamlessly automate the semantic versioning on your 
Git repository in a language and CI agnostic way.

Following the UNIX philosophy of "make each program do one thing well", it only handles publishing semver tags to your 
Git repository, no package publishing or any other features. 

All you need have is an initialized Git repository, a release branch (e.g., `main`) and a formatted commit history on 
that branch following the [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) specification. Many IDEs
support plugins to help in formatting your commit messages (e.g., [VSCode](https://marketplace.visualstudio.com/items?itemName=vivaxy.vscode-conventional-commits), [IntelliJ](https://plugins.jetbrains.com/plugin/13389-conventional-commit)).

> [!IMPORTANT]
> `go-semver-release` can only read **annotated** Git tags. If at some point you need to manually add a semver tag your
> repository, make sure it is annotated, otherwise the program will not be able to detect it.

## Install

If [Go](https://go.dev) is installed on your machine, you can install from source:

```bash
$ go install github.com/s0ders/go-semver-release@latest
$ go-semver-release --help
```

For cross-platform compatibility, you can use the generated [Docker image](https://hub.docker.com/r/soders/go-semver-release/tags):

```bash
$ docker pull s0ders/go-semver-release:latest
$ docker run --rm s0ders/go-semver-release --help
```

## Usage

Documentation about the CLI usage can be found [here](docs/usage.md).

## CI workflow examples

This tool is voluntarily agnostic of which CI tool is used with it. Examples of workflows with various CI tools can be found [here](docs/workflows.md).

## WIP
- [ ] PGP signing flag