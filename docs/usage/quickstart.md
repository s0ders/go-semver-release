# Quickstart

All you need to get started is:

* A Git repository available locally or remotely
* A commit history following the [Conventional Commits](https://www.conventionalcommits.org/en/) convention
* A configuration file inside the Git repository to version

&#x20;The following example configuration file suites most use cases:

```yaml
# <REPOSITORY_ROOT>/.semver.yaml
remote: true
remote-name: "origin"
git-name: "My Custom Robot Name"
git-email: "custom-robot@acme.com"
tag-prefix: "v"
rules:
  minor:
    - feat
  patch:
    - fix
    - perf
    - revert
branches:
  - name: "master"
  - name: "alpha"
    prerelease: true
```

Once the configuration file is saved inside the Git repository to version, the tool can be ran from inside a local environment or a CI runner as below:

```bash
$ go-semver-release release <REPOSITORY_PATH_OR_URL> --config <PATH_TO_CONFIG_FILE> [--dry-run, --verbose]
```
