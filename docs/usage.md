## Usage

The program only supports a local mode of execution. This means that it requires that the repository to version is
already present on the local file system.
The program takes the path of local Git repository, computes the next semver and tags the repository.

This mode is a good option security-wise since it lets you use the program without having to configure any kind of
access management since it does not require any access token.

Example:

```bash
$ go-semver-release local <REPOSITORY_PATH> --rules-path <PATH> --tag-prefix <PREFIX> \
                                            --release-branch <NAME> --dry-run --verbose
```

> [!TIP]
> Tag prefix can be changed during the lifetime of a repository (e.g., going from no prefix to `v`), this will
> **not** affect the semver tags history, the program will still be able to recognize previous semver tags.

For more informations about available flags and their default values, run:

```bash
$ go-semver-release <COMMAND> --help
```

## Tag prefix

## Release branch

## Release rules

Release rules define which commit type will trigger a release, and which type of release (i.e., `minor` or `patch`).

> [!IMPORTANT]
> Release type can only be `minor` or `patch`, `major` is reserved for breaking change only.

The following release rules are applied by default:
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

You can define custom release rules to suit your needs using a JSON file and by passing it to the program as
bellow.

```bash
$ go-semver-release local <REPOSITORY_PATH> --rules-path ./path/to/custom/rules.json
```

If a commit type (e.g., `chore`) is not specified in you rule file, it will not trigger any kind of release.

The following `type` are supported for release rules: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`,
`revert`, `style`, `test`.

## Dry-run

## Signing tag
