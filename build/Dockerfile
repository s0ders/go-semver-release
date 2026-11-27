# golang:1.26.11-bookworm
FROM golang@sha256:ea47ee45f1284d60f9cc09cb2c3bb4f508c256b640bd8bb30342ff13d1adcb52 AS builder

ARG APP_VERSION="v0.0.0+unknown"
ARG APP_BUILD_NUMBER="unknown"
ARG APP_COMMIT_HASH="unknown"

WORKDIR /app

# Download and verify dependencies against go.sum before copying the sources, so that this layer is cached.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-X github.com/s0ders/go-semver-release/v8/cmd.cmdVersion=${APP_VERSION} -X github.com/s0ders/go-semver-release/v8/cmd.buildNumber=${APP_BUILD_NUMBER} -X github.com/s0ders/go-semver-release/v8/cmd.buildCommitHash=${APP_COMMIT_HASH} -w -s" -o /app/app .

# alpine:3.22.2
FROM alpine@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]
