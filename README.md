# Go SemVer Release

This program aims to help automating version management of Git repository using simple Git annotated tags that follow the [SemVer](https://semver.org/) convention. 

To do so, this program fetches a repository commits and tags history and analyze each commit that follows the [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) convention. Depending on the commit type (e.g. `feat`, `test`, `chore`), the program will apply major, minor, patch or no release rule at all. These release rules define which part of the semantic version is bumped.

## Install

If [Go](https://go.dev) is installed on your machine, you can directly compile and install this program with `go install`:

```bash
$ go install github.com/s0ders/go-semver-release
$ go-semver-release --help
```

For more cross-platform compatibility, you can use the official [Docker image](https://hub.docker.com/r/soders/go-semver-release/tags):

```bash
$ docker pull soders/go-semver-release
$ docker run --rm soders/go-semver-release --help
```



## Prerequisites

There are only a few prerequisites for using this tool and getting the benefits it brings :

- The Git repository commits must follow the [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) convention, as it is what is used to compute the semantic version.
- The repository passed to the program (or action) must already be initialized (i.e. `HEAD` does not point to nothing)



## GitHub Actions

The `go-release-semver` program can easily be used inside your Actions pipeline. It takes the same parameters as those describe in the usage section bellow except for `--dry-run` (yet).

Bellow is an example usage of this action inside a GitHub Actions pipeline.

```yaml
jobs:
  go-semver-release:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Semver Release
      uses: s0ders/go-semver-release@0.11.1
      with:
        repository-url: "https://github.com/path/to/your/repo.git"
        tag-prefix: "v"
        branch : "release"
        token: ${{ secrets.ACCESS_TOKEN }}
```



## Usage

You must pass the Git URL of the repository to target using the `--url` flag:

```bash
$ go-semver-release --url [GIT URL]
```

Because the program needs to push tag to the remote, it must authenticate itself using a personal access token passed with `--token`:

```bash
$ go-semver-release --url ... --token [ACCESS TOKEN]
```

The release branch on which the commits are fetched is specified using `--branch`:

```bash
$ go-semver-release --url ... --token ... --branch release
```

Custom release rules are passed to the program using the `--rules`:

```bash
$ go-semver-release --url ... --token ... --rules [PATH TO RULES]
```

The program supports a dry-run mode that will only compute the next semantic version, if any, and will stop here without pushing any tag to the remote, dry-run is off by default:

```bash
$ go-semver-release --url ... --token ... --dry-run true
```

A custom prefix can be added to the tag name pushed to the remote, by default the tag name correspond to the SemVer (e.g. `1.2.3`) but you might want to use some prefix like `v` using `--tag-prefix`:

```bash
$ go-semver-release --url ... --token ... --tag-prefix v
```

**Note**: with the `--tag-prefix` flag is that you can change your tag prefix during the lifetime of your repository (e.g. going from `v` to `version-`) and this will **not** affect the way `go-semver-release` will fetch your semver tags history, meaning that the program will still be able to recognize semver tags made with your old-prefixes.



## Release Rules

Release rules define which commit type will trigger a release, and what type of release (i.e. major, minor, patch). By default, the program applies the following release rules:

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
        // {"type": "feat", "release": "major"},  // bad idea
        {"type": "perf", "release": "patch"}
    ]
}
```

The following `type` are supported for release rules: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`.

The following `release` types are supported for release rules: `major`, `minor`, `patch`.



## Work in progress

- [ ] Support custom release branch using a `--branch` flag
- [ ] Add support for `--dry-run` inside GitHub Actions