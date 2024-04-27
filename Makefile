build:
	go build -o bin/go-semver-release

test:
	go test -race -v -covermode=atomic ./...
vuln:
	govulncheck ./...

docker-build:
	docker build -f ./build/Dockerfile -t soders/go-semver-release .

action-lint:
	actionlint

.PHONY: build