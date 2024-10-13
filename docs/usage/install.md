# Install

### GitHub Releases

The preferred method of installation is via the releases generated by the CI workflow of the official GitHub repository:
https://github.com/s0ders/go-semver-release/releases/latest

Releases are available for various OS and architectures and come with generated [provenance](https://slsa.dev/spec/v1.0/provenance) of SLSA level 3 to avoid being tampered with, ensuring a strong layer of protection against supply chain attacks.

### Go

If [Go](https://go.dev) is installed on your machine, you can install from source:

```bash
$ go install github.com/s0ders/go-semver-release/v5@latest
$ go-semver-release --help
```

### Docker

A [Docker image](https://hub.docker.com/r/s0ders/go-semver-release/tags) is available as well:

```bash
$ docker pull s0ders/go-semver-release:latest
$ docker run --rm s0ders/go-semver-release --help
```


