# Configuration

> [!TIP]
> Validate your configuration file before running with `go-semver-release validate <CONFIG_FILE>`

## Basics

Configuration can be set via **flags**, **environment variables** (`GO_SEMVER_RELEASE_*`), or a **YAML file** (default: `.semver.yaml`).

**Precedence:** flags > environment variables > configuration file > defaults

```bash
# See all options and defaults
$ go-semver-release release --help
```

## Minimal Configuration

```yaml
# .semver.yaml
branches:
  - name: main
```

That's it. Sensible defaults will be used for everything else.

## Options Reference

### branches

Branches to read commit history from.

```yaml
branches:
  - name: main
  - name: rc
    prerelease: true
```

**Prerelease branches** produce versions like `1.0.0-rc.1`. When merged to a stable branch, they're automatically promoted to stable releases (e.g., `1.0.0`).

### rules

Which commit types trigger releases. Defaults:

| Release | Commit types |
|---------|--------------|
| minor | `feat` |
| patch | `fix`, `perf`, `revert` |

> [!NOTE]
> `major` releases are triggered by a `!` (such as `feat(api)!: ...`, `fix!: ...`), or `BREAKING CHANGE` in the commit message.

```yaml
rules:
  minor:
    - feat
  patch:
    - fix
    - perf
    - refactor
```

Valid types: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`

### monorepo

Version multiple projects in a single repository separately.

```yaml
monorepo:
  - name: api
    path: ./api/
  - name: web
    paths:        # Use 'paths' for multiple directories
      - ./web/
      - ./shared/
```

Tags are created as `<project>-<version>` (e.g., `api-v1.2.0`).

### tag-prefix

Prefix for version tags. Default: `v`

```yaml
tag-prefix: v    # Creates tags like v1.2.3
```

### build-metadata

Append build metadata to versions (e.g., `1.2.3+build.123`).

```yaml
build-metadata: $CI_JOB_ID
```

### gpg-key-path

Sign tags with a GPG key.

```yaml
gpg-key-path: /path/to/key.asc
```

> [!CAUTION]
> Use a dedicated key for CI, not your personal key. Ensure the file has restricted permissions (`chmod 600`).

### lightweight-tags

Create lightweight tags instead of annotated tags. Default: `false`

```bash
go-semver-release release --lightweight-tags
```

**When to use:**
- Migrating from tools like `semantic-release` that create lightweight tags
- When you don't need tag metadata (author, date, message)

> [!NOTE]
> The tool can read both lightweight and annotated tags regardless of this setting. This flag only affects tag *creation*.

### remote-name / access-token

For remote repositories. Default remote: `origin`

```bash
GO_SEMVER_RELEASE_ACCESS_TOKEN="secret" go-semver-release release https://github.com/user/repo.git
```

> [!WARNING]
> Never put access tokens in the config file. Use environment variables, flags or CI secrets.

### dry-run

Compute version without creating tags.

```bash
go-semver-release release --dry-run
```

### git-name / git-email

Author for annotated tags (ignored when `--lightweight-tags` is used). Defaults: `Go Semver Release` / `go-semver@release.ci`

### verbose

Enable detailed logging.

```bash
go-semver-release release --verbose
```

## Full Example

```yaml
# .semver.yaml
branches:
  - name: main
  - name: rc
    prerelease: true

rules:
  minor:
    - feat
  patch:
    - fix
    - perf
    - revert

tag-prefix: v
git-name: Release Bot
git-email: release@example.com # If tag signing is enabled, this should match the email associated with the key
```
