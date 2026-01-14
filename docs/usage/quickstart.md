# Quickstart

All you need to get started is:

* A Git repository available locally or remotely
* A commit history following the [Conventional Commits](https://www.conventionalcommits.org/en/) convention
* Optionally, a configuration file inside the Git repository to version

The following configuration file suits most use cases:

```yaml
# <REPOSITORY_ROOT>/.semver.yaml
rules:
  minor:
    - feat
  patch:
    - fix
    - perf
    - revert
branches:
  - name: "main"
  - name: "alpha"
    prerelease: true
```

Once the configuration file is saved inside the Git repository to version, the tool can be executed from inside a local environment or a CI runner as below:

```bash
$ go-semver-release release <REPOSITORY_PATH_OR_URL> --config <PATH_TO_CONFIG_FILE> [--dry-run, --verbose]
```
