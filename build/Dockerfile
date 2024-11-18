# golang:1.23.1-bookworm
FROM golang@sha256:1a5326b07cbab12f4fd7800425f2cf25ff2bd62c404ef41b56cb99669a710a83 AS builder

ARG APP_VERSION="v0.0.0+unknown"
ARG APP_BUILD_NUMBER="unknown"
ARG APP_COMMIT_HASH="unknown"

WORKDIR /app
COPY .. /app

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-X github.com/s0ders/go-semver-release/v6/cmd.cmdVersion=$APP_VERSION -X github.com/s0ders/go-semver-release/v6/cmd.buildNumber=$APP_BUILD_NUMBER  -X github.com/s0ders/go-semver-release/v6/cmd.buildCommitHash=$APP_COMMIT_HASH -w -s" -v -o app .

# alpine:3.20.3
FROM alpine@sha256:1e42bbe2508154c9126d48c2b8a75420c3544343bf86fd041fb7527e017a4b4a AS vulnscan

COPY --from=builder /app/app /app

COPY --from=aquasec/trivy:latest /usr/local/bin/trivy /usr/local/bin/trivy
RUN trivy rootfs --vuln-type os,library --severity MEDIUM,CRITICAL,HIGH --exit-code 1 --no-progress /

# alpine:3.20.3
FROM alpine@sha256:1e42bbe2508154c9126d48c2b8a75420c3544343bf86fd041fb7527e017a4b4a

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]