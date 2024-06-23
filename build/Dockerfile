# golang:1.22.4-bookworm
FROM golang@sha256:aec47843e52fee4436bdd3ce931417fa980e9055658b5142140925eea3044bea as builder

ARG APP_VERSION="v0.0.0+unknown"
ARG APP_BUILD_NUMBER="unknown"
ARG APP_COMMIT_HASH="unknown"

WORKDIR /app
COPY .. /app

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-X github.com/s0ders/go-semver-release/v2/cmd.version=$APP_VERSION -X github.com/s0ders/go-semver-release/v2/cmd.buildNumber=$APP_BUILD_NUMBER  -X github.com/s0ders/go-semver-release/v2/cmd.commitHash=$APP_COMMIT_HASH -w -s" -v -o app .

# alpine:3.20.0
FROM alpine@sha256:77726ef6b57ddf65bb551896826ec38bc3e53f75cdde31354fbffb4f25238ebd as vulnscan

COPY --from=builder /app/app /app

COPY --from=aquasec/trivy:latest /usr/local/bin/trivy /usr/local/bin/trivy
RUN trivy rootfs --vuln-type os,library --severity CRITICAL,HIGH --exit-code 1 --no-progress /

alpine:3.20.0
FROM alpine@sha256:77726ef6b57ddf65bb551896826ec38bc3e53f75cdde31354fbffb4f25238ebd

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]