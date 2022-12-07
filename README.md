# Go Semver Release

This program aims to help automating version management of Git repository using simple Git annotated tags that follow the [SemVer](https://semver.org/) convention. 

To do so, this program fetches a repository commits and tags history and analyze each commit that follows the [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) convention. Depending on the commit type (e.g. `feat`, `test`, `chore`), the program will apply major, minor, patch or no release rule at all. These release rules define which part of the SemVer is bumped.

## Install

If [Go](https://go.dev) is installed on your machine, you can directly compile and install this program by using `go install`:

```bash
$ go install github.com/s0ders/go-semver-release
$ go-semver-release --help
```

Else, for more cross-platform compatibility, you can use the [Docker image](https://hub.docker.com/r/soders/go-semver-release/tags):

```bash
$ docker pull soders/go-semver-release
$ docker run --rm soders/go-semver-release --help
```



## Usage

You must pass the Git URL of the repository to target using the `--url` flag:

```bash
$ go-semver-release --url GIT_REPOSITORY_URL
```

Because the program needs to push tag to the remote, it must authenticate itself using a personal access token that you pass using the `--token` flag:

```bash
$ go-semver-release --url ... --token MY_SECRET_TOKEN
```

The program supports a dry-run mode that will only compute the next SemVer, if any, and will stop here without pushing any tag to the remote:

```bash
$ go-semver-release --url ... --token ... --dry-run true
```

Custom release rules are passed to the program using the `--rules` flag:

```bash
$ go-semver-release --url ... --token ... --rules PATH_TO_RULES
```

Custom prefix can be added to the tag name pushed to the remote, by default the tag name correspond to the SemVer (e.g. `1.2.3`) but you might want to use some prefix like `v` using the `--tag-prefix` flag:

```bash
$ go-semver-release --url ... --token ... --tag-prefix v
```



One important thing to note with the `--tag-prefix` flag is that you can change your tag prefix during the lifetime of your repository (e.g. going from `v` to `version-`) and this will **not** affect the way `go-semver-release` will fetch your semver tags history, meaning that the program will still be able to recognize semver tags made with your old-prefixes.



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

You can define custom release rules to suit your needs using a JSON file and by passing it to the program:

```json
{
    "releaseRules": [
        {"type": "feat", "release": "major"},
        {"type": "fix", "release": "patch"},
        {"type": "perf", "release": "patch"}
    ]
}
```

The following `type` are supported for release rules: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`.

The following `release` types are supported for release rules: `major`, `minor`, `patch`.



## Work in progress

- [ ] Publish a Docker container GitHub Action via the CI/CD
- [ ] Add an asciinema demonstration to the docs

