build:
	GOOS=linux GOARCH=amd64 go build -o bin/go-semver-release-linux-amd64 main.go

test:
	go test -race ./...

vuln:
	govulncheck ./...

docker-build:
	docker build -f ./build/Dockerfile -t soders/go-semver-release .