# Go Semver Release

<p align="center">
  <img src=".gitbook/assets/gop.png" alt="Go Semver Release Logo" width="230">
  <br><br>
  <a href="https://github.com/avelino/awesome-go"><img alt="Mentioned in Awesome Go" src="https://awesome.re/mentioned-badge.svg"></a>
  <a href="https://img.shields.io/github/v/tag/s0ders/go-semver-release?label=Version&color=bb33ff"><img alt="GitHub Tag" src="https://img.shields.io/github/v/tag/s0ders/go-semver-release?label=Version\&color=bb33ff"></a>
  <a href="https://img.shields.io/github/go-mod/go-version/s0ders/go-semver-release"><img alt="GitHub go.mod Go version" src="https://img.shields.io/github/go-mod/go-version/s0ders/go-semver-release"></a>
  <a href="https://pkg.go.dev/github.com/s0ders/go-semver-release/v5"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/s0ders/go-semver-release.svg"></a>
  <a href="https://img.shields.io/github/actions/workflow/status/s0ders/go-semver-release/main.yaml?label=CI"><img alt="GitHub Actions Workflow Status" src="https://img.shields.io/github/actions/workflow/status/s0ders/go-semver-release/main.yaml?label=CI"></a>
  <a href="https://goreportcard.com/report/github.com/s0ders/go-semver-release/v5"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/s0ders/go-semver-release/v5"></a>
  <a href="https://app.codecov.io/github/s0ders/go-semver-release"><img alt="Codecov" src="https://img.shields.io/codecov/c/github/s0ders/go-semver-release?label=Coverage"></a>
  <a href="https://github.com/s0ders/go-semver-release/blob/main/LICENSE.md"><img alt="GitHub License" src="https://img.shields.io/github/license/s0ders/go-semver-release?label=License"></a>
  <a href="https://www.bestpractices.dev/projects/8877"><img alt="OpenSSF Best Practices" src="https://www.bestpractices.dev/projects/8877/badge"></a>
</p>

Go Semver Release is a CLI program designed to automate versioning of Git repository by analyzing their [formatted commit history](https://www.conventionalcommits.org) and tagging them with the right [SemVer](https://semver.org/spec/v2.0.0.html) number.

This documentation is also available at [https://go-semver-release.akira.sh](https://go-semver-release.akira.sh)&#x20;

## Features

* 🏷️ Automatic semantic versioning of your Git repository via annotated Git tags
* 🌐 Local or remote mode of execution (local removes the need for secret token)
* 🌴 Support for multiple release branch, prerelease and build metadata
* 🗂️ Support monorepo (i.e. multiple projects inside a single repository, all versioned separately)
* ⚙️ Custom tag prefix
* 📝 Tag signature using GPG

## Motivation

This project was built to create a lightweight and simple tool to automate the semantic versioning on your Git repository in a language and CI agnostic way by strictly following the Semantic Versioning and Conventional Commit convention.

Following the UNIX philosophy of "make each program do one thing well", it only handles publishing SemVer tags to your Git repository, no package publishing or any other features.

All you need to have is an initialized Git repository, a release branch (e.g., `main`) and a formatted commit history on that branch following the [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) specification. Many IDEs support plugins that help formatting messages (e.g., [VSCode](https://marketplace.visualstudio.com/items?itemName=vivaxy.vscode-conventional-commits), [IntelliJ](https://plugins.jetbrains.com/plugin/13389-conventional-commit)).

> [!NOTE]
> This program can only read annotated Git tags. If at some point you need to manually add a SemVer tag to your repository, make sure it is annotated, otherwise the program will not be able to detect it.

## How is this different from \<insert\_another\_tool> ?

Other tools exist to version software using semantic versions such as [semantic-release](https://github.com/semantic-release/semantic-release). Go Semver Release focuses on versioning only, no package publishing, release log generation or other features.

If you want a simple tool that handle the generation of the next semantic version tag for your project, you are at the right place. This allows the program to work with minimal dependencies, limit surface of attack and work faster.

## Documentation

### Usage

* [Install](usage/install.md)
* [Quickstart](usage/quickstart.md)
* [Configuration](usage/configuration.md)
* [Output](usage/output.md)
* [How it works](usage/how-it-works.md)

### Recipes

* [Workflow examples](recipes/workflow-examples.md)

<br>
<hr>

Project's illustration designed by [@TristanDacross](https://github.com/TristanDacross)