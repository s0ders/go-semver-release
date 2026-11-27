# golang:1.26.11-bookworm
FROM golang@sha256:ea47ee45f1284d60f9cc09cb2c3bb4f508c256b640bd8bb30342ff13d1adcb52 AS builder

ARG APP_VERSION="v0.0.0+unknown"
ARG APP_BUILD_NUMBER="unknown"
ARG APP_COMMIT_HASH="unknown"

WORKDIR /app
COPY .. /app

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-X github.com/s0ders/go-semver-release/v7/cmd.cmdVersion=$APP_VERSION -X github.com/s0ders/go-semver-release/v7/cmd.buildNumber=$APP_BUILD_NUMBER  -X github.com/s0ders/go-semver-release/v7/cmd.buildCommitHash=$APP_COMMIT_HASH -w -s" -v -o app .

# alpine:3.22.2
FROM alpine@sha256:85f2b723e106c34644cd5851d7e81ee87da98ac54672b29947c052a45d31dc2f AS vulnscan

COPY --from=builder /app/app /app

COPY --from=aquasec/trivy:latest /usr/local/bin/trivy /usr/local/bin/trivy
RUN trivy rootfs --vuln-type os,library --severity MEDIUM,CRITICAL,HIGH --exit-code 1 --no-progress /

# alpine:3.22.2
FROM alpine@sha256:85f2b723e106c34644cd5851d7e81ee87da98ac54672b29947c052a45d31dc2f

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]