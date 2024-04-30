# golang:1.22.2
FROM golang@sha256:d5302d40dc5fbbf38ec472d1848a9d2391a13f93293a6a5b0b87c99dc0eaa6ae as builder

ARG APP_VERSION="v0.0.0+unknown"
ARG APP_BUILD_NUMBER="unknown"
ARG APP_COMMIT_HASH="unknown"

WORKDIR /app
COPY .. /app

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-X github.com/s0ders/go-semver-release/v2/cmd.version=$APP_VERSION -X github.com/s0ders/go-semver-release/v2/cmd.buildNumber=$APP_BUILD_NUMBER  -X github.com/s0ders/go-semver-release/v2/cmd.commitHash=$APP_COMMIT_HASH -w -s" -v -o app .

# alpine:3.19.1
FROM alpine@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]