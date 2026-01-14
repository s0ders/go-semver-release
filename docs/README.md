# Go Semver Release

<p align="center">
  <img src=".gitbook/assets/gop.png" alt="Go Semver Release Logo" width="230">
  <br><br>
  <a href="https://github.com/avelino/awesome-go"><img alt="Mentioned in Awesome Go" src="https://awesome.re/mentioned-badge.svg"></a>
  <a href="https://img.shields.io/github/v/tag/s0ders/go-semver-release?label=Version&color=bb33ff"><img alt="GitHub Tag" src="https://img.shields.io/github/v/tag/s0ders/go-semver-release?label=Version&color=bb33ff"></a>
  <a href="https://img.shields.io/github/actions/workflow/status/s0ders/go-semver-release/main.yaml?label=CI"><img alt="GitHub Actions Workflow Status" src="https://img.shields.io/github/actions/workflow/status/s0ders/go-semver-release/main.yaml?label=CI"></a>
  <a href="https://goreportcard.com/report/github.com/s0ders/go-semver-release/v7"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/s0ders/go-semver-release/v7"></a>
  <a href="https://app.codecov.io/github/s0ders/go-semver-release"><img alt="Codecov" src="https://img.shields.io/codecov/c/github/s0ders/go-semver-release?label=Coverage"></a>
</p>

Analyzes your commit history and creates the next [SemVer](https://semver.org) tag automatically.

## Features

* **Zero configuration** — works out of the box with sensible defaults
* **Support for monorepo** — version multiple projects independently in the same repository
* **Prerelease branches** — `1.0.0-rc.1`, `1.0.0-beta.2`
* **GPG signing** — sign produced tags
* **CI-agnostic** — works with GitHub, GitLab or locally

## Quickstart 

```bash
# If Go is installed on your machine, else, download the latest release
$ go install github.com/s0ders/go-semver-release/v7@latest

$ cd ~/my/git/repository

# Run (dry-run to see what would happen)
$ go-semver-release release --dry-run
```

## Documentation

### Usage

* [Install](usage/install.md)
* [Quickstart](usage/quickstart.md)
* [Configuration](usage/configuration.md)
* [Output](usage/output.md)

### Miscellaneous

* [Workflow examples](recipes/workflow-examples.md)
* [How it works](miscellaneous/how-it-works.md)
* [Benchmark](miscellaneous/benchmark.md)

## How is this different from \<insert\_another\_tool> ?

Other tools exist to version software using semantic versions such as [semantic-release](https://github.com/semantic-release/semantic-release). 

Go Semver Release focuses on versioning only. No package publishing or other feature requiring extra configuration. 

To each tool its responsibilities and these are best left to programs such as [Go Releaser](https://goreleaser.com/) 
which you may use in combination with Go Semver Release.

<hr>

Project's illustration designed by [@TristanDacross](https://github.com/TristanDacross)
