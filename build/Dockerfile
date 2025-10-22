# golang:1.24.9-bookworm
FROM golang@sha256:65b6fbbd07ba05c79dc51889f6d9c62e4c8071b02593f2c8b49a381de3a0a3fe AS builder

ARG APP_VERSION="v0.0.0+unknown"
ARG APP_BUILD_NUMBER="unknown"
ARG APP_COMMIT_HASH="unknown"

WORKDIR /app
COPY .. /app

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-X github.com/s0ders/go-semver-release/v6/cmd.cmdVersion=$APP_VERSION -X github.com/s0ders/go-semver-release/v6/cmd.buildNumber=$APP_BUILD_NUMBER  -X github.com/s0ders/go-semver-release/v6/cmd.buildCommitHash=$APP_COMMIT_HASH -w -s" -v -o app .

# alpine:3.22.2
FROM alpine@sha256:85f2b723e106c34644cd5851d7e81ee87da98ac54672b29947c052a45d31dc2f AS vulnscan

COPY --from=builder /app/app /app

COPY --from=aquasec/trivy:latest /usr/local/bin/trivy /usr/local/bin/trivy
RUN trivy rootfs --vuln-type os,library --severity MEDIUM,CRITICAL,HIGH --exit-code 1 --no-progress /

# alpine:3.22.2
FROM alpine@sha256:85f2b723e106c34644cd5851d7e81ee87da98ac54672b29947c052a45d31dc2f

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]