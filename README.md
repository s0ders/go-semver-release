<img alt="go version badge" src="https://img.shields.io/github/go-mod/go-version/s0ders/go-semver-release"> <img alt="github actions badge" src="https://github.com/s0ders/go-semver-release/actions/workflows/main.yaml/badge.svg"> <img alt="go report card" src="http://goreportcard.com/badge/github.com/s0ders/go-semver-release"> <img alt="license badge" src="https://img.shields.io/github/license/s0ders/go-semver-release"> <img alt="gitleaks badge" src="https://img.shields.io/badge/protected%20by-gitleaks-blue"> 

# Go SemVer Release

Go program designed to automate versioning of Git repository by analyzing their formatted commit history and tagging them with the right semver number. This program can be used directly via its CLI or its corresponding [GitHub Action](https://github.com/marketplace/actions/go-semver-release).

<ul>
    <li><a href="#Motivations">Motivations</a></li>
    <li><a href="#Install">Install</a></li>
    <li><a href="#Usage">Usage</a></li>
    <li><a href="#github-actions">Github Actions</a></li>
    <li><a href="#release-rules">Release Rules</a></li>
</ul>


## Motivation

Handling a Git repository versions can be done using well-thought convention such as [SemVer](https://semver.org/) so that your API consumers know when a non-retro-compatible change is introduced in your API. Building on that, versioning automation can be achieved by using formated commits following the [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) convention. 

This project was built to create a lightweight and simple tool to seamlessly automate versioning on your Git repository. Following the UNIX philosophy of "make each program do one thing well", it only handles publishing semver tags to your Git repository, no package publishing or any other features. 

The Docker image merely weight `7Mb` and the Go program inside will compute your semver tag in seconds, no matter the size of your commit history.

This tool aims to integrate semantic versioning automation in such a way that, all you have to do is:

- Choose a release branch (e.g. `main`, `release`)

- Take care to format commits on that branch by following the [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) convention, which many IDEs plugins offers to do seamlessly (e.g. [VSCode](https://marketplace.visualstudio.com/items?itemName=vivaxy.vscode-conventional-commits), [IntelliJ](https://plugins.jetbrains.com/plugin/13389-conventional-commit))

> **Note**: `go-semver-release` can only read **annotated** Git tags, so if you plan on only using it in dry-run mode to then use its output to tag your repository with an other action, make sure the tag you are pushing is annotated, otherwise the program will not be able to detect it.

## Install

If [Go](https://go.dev) is installed on your machine, you can install from source using `go install`:

```bash
$ go install github.com/s0ders/go-semver-release/cmd/go-semver-release@latest
$ go-semver-release --help
```

For cross-platform compatibility, you can use the generated [Docker image](https://hub.docker.com/r/soders/go-semver-release/tags):

```bash
$ docker pull soders/go-semver-release
$ docker run --rm soders/go-semver-release --help
```

Verify that the downloaded image has not be tampered using [Cosign](https://github.com/sigstore/cosign):
```bash
$ PUB_KEY=https://raw.githubusercontent.com/s0ders/go-semver-release/main/cosign.pub
$ cosign verify --key $PUB_KEY soders/go-semver-release:v1.4.5
```


## Prerequisites

There are only a few prerequisites for using this tool and getting the benefits it brings :

- The Git repository commits must follow the [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) convention, as it is what is used to compute the semantic version.
- The repository passed to the program (or action) must already be initialized (i.e. Git `HEAD` does not point to nothing)



## Usage

You must pass the Git URL of the repository to target using the `--url` flag:

```bash
$ go-semver-release --url <GIT URL>
```

Because the program needs to push tag to the remote, it must authenticate itself using a personal access token passed with `--token`:

```bash
$ go-semver-release [...] --token <ACCESS TOKEN>
```

The release branch on which the commits are fetched is specified using `--branch`:

```bash
$ go-semver-release [...] --branch <BRANCH NAME>
```

Custom release rules are passed to the program using the `--rules`:

```bash
$ go-semver-release [...] --rules <PATH TO RULES>
```

The program supports a dry-run mode that will only compute the next semantic version, if any, and will stop here without pushing any tag to the remote, dry-run is `false` by default:

```bash
$ go-semver-release [...] --dry-run true
```

A custom prefix can be added to the tag name pushed to the remote, by default the tag name correspond to the SemVer (e.g. `1.2.3`) but you might want to use some prefix like `v` using `--tag-prefix`:

```bash
$ go-semver-release [...] --tag-prefix <PREFIX>
```

> **Note**: You can change your tag prefix during the lifetime of your repository (e.g. going from none to `v`) and this will **not** affect your semver tags history, meaning that the program will still be able to recognize semver tags made with your old-prefixes, if any. There are no limitation to how many time you can change your tag prefix during the lifetime of your repository.



## GitHub Actions

### Inputs

The action takes the same parameters as those defined in the <a href="#Usage">usage</a> section. Note that the `--dry-run` needs to be passed as a string inside your YAML work-flow due to how Github Actions works.

### Outputs

The action generate a two outputs 
- `SEMVER`, the computed semver or the current one if no new were computed, prefixed with the given `tag-prefix` if any;
- `NEW_RELEASE`, whether a new semver was computed or not.

### Example Workflow

Bellow is an example of this action inside a GitHub Actions pipeline.

```yaml
jobs:
  go-semver-release:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Semver Release
      uses: s0ders/go-semver-release@v1.4.5
      with:
        repository-url: 'https://github.com/path/to/your/repo.git'
        tag-prefix: 'v'
        branch: 'release'
        dry-run: 'false'
        token: ${{ secrets.ACCESS_TOKEN }}
```

## Release Rules

Release rules define which commit type will trigger a release, and what type of release (i.e. major, minor, patch). **By default**, the program applies the following release rules:

```json
{
    "releaseRules": [
        {"type": "feat", "release": "minor"},
        {"type": "perf", "release": "minor"},
        {"type": "fix", "release": "patch"}
    ]
}
```

You can define custom release rules to suit your needs using a JSON file and by passing it to the program as bellow. Be careful with release rules though, especially major ones, as their misuse might can easily make you loose the benefits of using a semantic version number.

```json
{
    "releaseRules": [
        {"type": "feat", "release": "minor"},
        {"type": "perf", "release": "patch"},
        {"type": "refactor", "release": "patch"},
        {"type": "fix", "release": "patch"}
    ]
}
```

The following `type` are supported for release rules: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`.

The following `release` types are supported for release rules: `major`, `minor`, `patch`.

