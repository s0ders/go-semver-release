# golang:1.22.4-bookworm
FROM golang@sha256:2fe82a3f3e006b4f2a316c6a21f62b66e1330ae211d039bb8d1128e12ed57bf1 AS builder

ARG APP_VERSION="v0.0.0+unknown"
ARG APP_BUILD_NUMBER="unknown"
ARG APP_COMMIT_HASH="unknown"

WORKDIR /app
COPY .. /app

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-X github.com/s0ders/go-semver-release/v4/cmd.version=$APP_VERSION -X github.com/s0ders/go-semver-release/v4/cmd.buildNumber=$APP_BUILD_NUMBER  -X github.com/s0ders/go-semver-release/v4/cmd.commitHash=$APP_COMMIT_HASH -w -s" -v -o app .

# alpine:3.20.1
FROM alpine@sha256:b89d9c93e9ed3597455c90a0b88a8bbb5cb7188438f70953fede212a0c4394e0 AS vulnscan

COPY --from=builder /app/app /app

COPY --from=aquasec/trivy:latest /usr/local/bin/trivy /usr/local/bin/trivy
RUN trivy rootfs --vuln-type os,library --severity MEDIUM,CRITICAL,HIGH --exit-code 1 --no-progress /

# alpine:3.20.1
FROM alpine@sha256:b89d9c93e9ed3597455c90a0b88a8bbb5cb7188438f70953fede212a0c4394e0

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]