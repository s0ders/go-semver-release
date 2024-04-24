build:
	go build -o bin/go-semver-release

test:
	go test -race ./...

vuln:
	govulncheck ./...

docker-build:
	docker build -f ./build/Dockerfile -t soders/go-semver-release .

.PHONY: build