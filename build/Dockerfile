FROM golang:1.22 as builder

ARG APP_VERSION="v0.0.0+unknown"
ARG APP_BUILD_NUMBER="unknown"
ARG APP_COMMIT_HASH="unknown"

WORKDIR /app
COPY .. /app

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-X github.com/s0ders/go-semver-release/v2/cmd.version=$APP_VERSION -X github.com/s0ders/go-semver-release/v2/cmd.buildNumber=$APP_BUILD_NUMBER  -X github.com/s0ders/go-semver-release/v2/cmd.commitHash=$APP_COMMIT_HASH -w -s" -v -o app .

FROM alpine:3.19

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]