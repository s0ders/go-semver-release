alias tc := test-coverage

ext := if os_family() == "windows" { ".exe" } else { "" }
outPath := "./bin/go-semver-release"

# Default values overridden in CI
appVersion := "v0.0.0+local"
buildNumber := "local"
commitHash := "local"

importPath := "github.com/s0ders/go-semver-release/v6/"
ldFlags := "-X " + importPath + "cmd.cmdVersion=" + appVersion + " -X " + importPath + "cmd.buildNumber=" + buildNumber + " -X " + importPath + "cmd.buildCommitHash=" + commitHash + " -w -s"

# gotestsum v1.13.0
tests:
	go run gotest.tools/gotestsum@c4a0df2e75a225d979a444342dd3db752b53619f --hide-summary=skipped -- -shuffle=on -race -covermode=atomic ./...

test-coverage: clean-coverage
    go test -coverprofile cover.out ./...
    go tool cover -html cover.out -o cover.html

clean-coverage:
    rm -f cover.out cover.html

test name:
    go test -tags testing '-run=^{{name}}$' -race ./...

build: clean
	go build -ldflags="{{ldFlags}}" -o {{outPath}}{{ext}} .

cross-platform-build: clean
    GOARCH=amd64 GOOS=darwin go build -ldflags="{{ldFlags}}" -o {{outPath}}-amd64-darwin
    GOARCH=arm64 GOOS=darwin go build -ldflags="{{ldFlags}}" -o {{outPath}}-arm64-darwin
    GOARCH=amd64 GOOS=linux go build -ldflags="{{ldFlags}}" -o {{outPath}}-amd64-linux
    GOARCH=arm64 GOOS=linux go build -ldflags="{{ldFlags}}" -o {{outPath}}-arm64-linux
    GOARCH=amd64 GOOS=windows go build -ldflags="{{ldFlags}}" -o {{outPath}}-amd64-win.exe

clean:
    rm -rf ./bin/*

lint:
	golangci-lint run ./...
	gocyclo -over 15 .

vuln:
	@govulncheck ./...

docker-build:
	docker build -f ./build/Dockerfile -t soders/go-semver-release:local .

action-lint:
	@actionlint

install-tooling:
    go install github.com/rhysd/actionlint/cmd/actionlint@latest
    go install golang.org/x/vuln/cmd/govulncheck@latest
    go install github.com/fzipp/gocyclo/cmd/gocyclo@latest