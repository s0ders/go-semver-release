# Benchmark

The following showcases performances benchmarks of Go Semver Release against other such tools. Theses benchmarks were realized using [hyperfine ](https://github.com/sharkdp/hyperfine)in GitHub Action [runner ](https://docs.github.com/en/actions/using-github-hosted-runners/using-github-hosted-runners/about-github-hosted-runners#standard-github-hosted-runners-for-public-repositories)with the following configuration:

* OS: Ubuntu 24.04
* Processor (CPU): 4
* Memory: 16 GB

Each program was executed 10 times (avoids statistic outliers) computing the latest semantic version of a [sample repository](https://github.com/s0ders/big-sample-repo) of 10,000 commits on a single "main" branch without any prior tag:

| Program                                                                  | Time (mean ± σ)   | Range (min ... max) |
| ------------------------------------------------------------------------ | ----------------- | ------------------- |
| [Go Semver Release](https://github.com/s0ders/go-semver-release)         | 1.127 s ± 0.244 s | 0.986 s ... 1.600 s |
| [Semantic Release](https://github.com/semantic-release/semantic-release) | 5.150 s ± 0.046 s | 5.041 s ... 5.207 s |

