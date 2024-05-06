# How to contribute
I'm glad you are considering contributing to this project. Bellow are the guidelines you must follow to do so.

## Change scope
This project focuses solely on versioning Git repositories and does not aim to do anything more than that.
Modifications that include anything non-related to this topic will not get approved.

## Building and tooling

To build this project you need the [Go standard toolchain](https://go.dev/dl/) and [just](https://just.systems/man/en/).
You can read the Makefile to see what tools are necessary to run each target (e.g., staticcheck, govulncheck, etc.)

## Testing
This project maintains a test coverage that is >= 80%. Please test extensively any new addition you want to make using
Go testing features as well as `github.com/stretchr/testify` package if you feel it might help write better tests.
After adding your modifications, please run all the project unit tests as bellow to ensure everything still works as
intented:
```go
// Run the following on project's root
$ make test
```

## Submitting changes

All changes must be submitted via a [pull request](https://github.com/s0ders/go-semver-release/pulls). To do so, fork
this repository and submit a new pull request.


## Coding conventions



Reading the project's code should be enough to get an idea of what are its conventions. If you need more extensive
documentations, please read:
- [The Uber Go Style Guide](https://github.com/uber-go/guide)
- [Google Go Style Best Practices](https://google.github.io/styleguide/go/best-practices.html)

> [!TIP]
> You don't need to read everything from these documents. Though they are really intersting, some features they describe
> are not in use in this project.

Besides that, all code must be formatted using `gofumpt` and imports formatted with `goimports`.