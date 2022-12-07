# Go Semver Release

This program aims to help automating version management of Git repository using simple Git annotated tags that follow the [Semver](https://semver.org/) convention. 

To do so, this program fetches a repository commits and tags history and analyze each commit that follows the [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) convention. Depending on the commit type (e.g. `feat`, `test`, `chore`), the program will apply major, minor, patch or no release rule at all. These release rules define which part of the Semver is bumped.

## Install

You can either compile the main binary on your architecture as bellow:

```bash
$ go install github.com/s0ders/go-semver-release
$ go-semver-release --help
```

Or use the Docker image:

```bash
$ docker pull soders/go-semver-release
$ docker run --rm soders/go-semver-release --help
```



## Usage

You must pass the Git URL of the repository to target using the `--url` flag:

```bash
$ go-semver-release --url GIT_REPOSITORY_URL
```

Because the program needs to push tag to the remote, if must authenticate itself using a personal access token that you pass using the `--token` flag:

```bash
$ go-semver-release --url ... --token MY_SECRET_TOKEN
```

The program supports a dry-run mode that will only compute the next Semver, if any, and will stop here without pushing any tag to the remote:

```bash
$ go-semver-release --url ... --token ... --dry-run true
```

Custom release rules are passed to the program using the `--rules` flag:

```bash
$ go-semver-release --url ... --token ... --rules PATH_TO_RULES
```



## Release Rules

Release rules define which commit type will trigger a release, and what type of release (i.e. major, minor, patch). By default, the program applies the following release rules bellow, which will trigger a minor release every "feat" commit, and a patch release for every "fix".

```json
{
    "releaseRules": [
        {"type": "feat", "release": "minor"},
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