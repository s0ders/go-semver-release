ifeq ($(OS),Windows_NT)
    detected_OS := Windows
    RM := powershell -Command "Remove-Item -Path"
    EXT := .exe
else
    detected_OS := $(shell uname -s)
    RM := rm -f
    EXT :=
endif

build:
	go build -ldflags="-X github.com/s0ders/go-semver-release/v2/cmd.cliVersion=v0.0.0+local" -o bin/go-semver-release$(EXT) .

clean:
	$(RM) bin/go-semver-release$(EXT)

test:
	go test -race -v -covermode=atomic ./...

vuln:
	govulncheck ./...

docker-build:
	docker build -f ./build/Dockerfile --build-arg="APP_VERSION=v0.0.0+local"  -t soders/go-semver-release .

action-lint:
	actionlint

.PHONY: build clean