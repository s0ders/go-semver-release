# golang:1.23.1-bookworm
FROM golang@sha256:adee809c2d0009a4199a11a1b2618990b244c6515149fe609e2788ddf164bd10 AS builder

ARG APP_VERSION="v0.0.0+unknown"
ARG APP_BUILD_NUMBER="unknown"
ARG APP_COMMIT_HASH="unknown"

WORKDIR /app
COPY .. /app

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-X github.com/s0ders/go-semver-release/v5/cmd.version=$APP_VERSION -X github.com/s0ders/go-semver-release/v5/cmd.buildNumber=$APP_BUILD_NUMBER  -X github.com/s0ders/go-semver-release/v5/cmd.commitHash=$APP_COMMIT_HASH -w -s" -v -o app .

# alpine:3.20.3
FROM alpine@sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d AS vulnscan

COPY --from=builder /app/app /app

COPY --from=aquasec/trivy:latest /usr/local/bin/trivy /usr/local/bin/trivy
RUN trivy rootfs --vuln-type os,library --severity MEDIUM,CRITICAL,HIGH --exit-code 1 --no-progress /

# alpine:3.20.3
FROM alpine@sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]