# Benchmark

The following compares the performances of Go Semver Release with other tools computing the next semantic version of a repository from its [Conventional Commits](https://www.conventionalcommits.org/) history.

Each program computed the next version of a [sample repository](https://github.com/s0ders/big-sample-repo) of 10,016 commits (about 3,400 `feat`, 3,300 `fix` and 3,300 `chore`) on a single `main` branch, without any prior tag. Each program was run 10 times, after 2 warmup runs, using [hyperfine](https://github.com/sharkdp/hyperfine).

| Program                                                                                    | Language | Time (mean ± σ)  | Range (min … max) | Peak memory | Next version |
| ------------------------------------------------------------------------------------------ | -------- | ---------------- | ----------------- | ----------- | ------------ |
| [svu](https://github.com/caarlos0/svu) 3.4.4                                               | Go       | 96 ms ± 6 ms     | 88 ms … 106 ms    | 14 MB       | `v0.1.0`     |
| [cocogitto](https://github.com/cocogitto/cocogitto) 7.0.0                                  | Rust     | 103 ms ± 4 ms    | 100 ms … 111 ms   | 20 MB       | `0.1.0`      |
| **[Go Semver Release](https://github.com/s0ders/go-semver-release) 8.0.6**                 | **Go**   | **109 ms ± 5 ms** | **105 ms … 120 ms** | **34 MB** | **`0.1.0`**  |
| [Commitizen](https://github.com/commitizen-tools/commitizen) 4.20.1                        | Python   | 289 ms ± 9 ms    | 282 ms … 306 ms   | 50 MB       | `0.1.0`      |
| [go-semantic-release](https://github.com/go-semantic-release/semantic-release) 2.31.0      | Go       | 525 ms ± 22 ms   | 508 ms … 580 ms   | 61 MB       | `0.1.0`      |
| [git-cliff](https://github.com/orhun/git-cliff) 2.14.2                                     | Rust     | 728 ms ± 16 ms   | 709 ms … 760 ms   | 51 MB       | `0.1.0`      |
| [semantic-release](https://github.com/semantic-release/semantic-release) 26.0.1            | Node.js  | 840 ms ± 9 ms    | 828 ms … 852 ms   | 287 MB      | `1.0.0`      |
| [Python Semantic Release](https://github.com/python-semantic-release/python-semantic-release) 10.7.0 | Python | 2,414 ms ± 33 ms | 2,363 ms … 2,465 ms | 72 MB | `1.0.0` |

Peak memory is the mean, over the 10 runs, of the peak resident memory reported by hyperfine. semantic-release and Python Semantic Release start a repository without any prior release at `1.0.0` by convention, where the other tools start at `0.1.0`.

## Reading the results

These tools do not all perform the same work to compute a version:

* **Go Semver Release** also resolves the release branches and prerelease versions configured for the repository, and supports monorepositories. Remote repositories, given by URL, are cloned first, which adds the clone time; local repositories, as here, are opened in place.
* **svu** only computes a version, without configuration for branches, prereleases or monorepositories.
* **semantic-release** was restricted to its commit analyzer plugin; its default configuration would also generate release notes and publish to npm and GitHub. **go-semantic-release** also generates a changelog, which cannot be disabled.

For comparison, the previous release benchmarked under the same conditions, Go Semver Release 8.0.5, took 471 ms ± 15 ms (40 MB): 8.0.6 is 4.3 times faster, by opening local repositories in place instead of cloning them, and by keeping packfiles open instead of reopening them for every object read.

## Methodology

The benchmark was run on November 28th, 2026 with the following configuration:

* OS: Ubuntu 24.04 (WSL 2, Linux 6.18)
* Processor (CPU): AMD Ryzen 7 7800X3D, 8 cores / 16 threads
* Memory: 15 GB

Every program was installed from its official release, verified against its published checksum (and, for Go Semver Release, its SLSA provenance). Each program ran in its own copy of the sample repository, so that no program could be affected by the files another one creates, and with network access blocked, so that timings only cover local work (go-semantic-release plugins were downloaded beforehand). Every program was checked to leave its copy unmodified. These results cannot be compared with previous versions of this page, which were measured on a GitHub Actions runner.

The following commands were benchmarked, from the root of the sample repository:

| Program                 | Command                                                                                             |
| ----------------------- | --------------------------------------------------------------------------------------------------- |
| Go Semver Release       | `go-semver-release release . --dry-run --branches '[{"name":"main"}]'`                              |
| svu                     | `svu next`                                                                                          |
| cocogitto               | `cog bump --auto --dry-run`                                                                         |
| Commitizen              | `cz --config cz.toml bump --get-next --yes` (`scm` version provider, `v$version` tag format)        |
| go-semantic-release     | `semantic-release --dry --no-ci --provider git --allow-initial-development-versions`                |
| git-cliff               | `git-cliff --bumped-version`                                                                        |
| semantic-release        | `semantic-release --dry-run --no-ci --branches main --plugins @semantic-release/commit-analyzer --repository-url file://$PWD` |
| Python Semantic Release | `semantic-release version --print`                                                                  |
