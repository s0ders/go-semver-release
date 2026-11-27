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

# The binary is static: the runtime image only needs CA certificates and a writable /tmp, which distroless provides
# without any shell or package manager (and their vulnerabilities).
# gcr.io/distroless/static-debian12:latest
FROM gcr.io/distroless/static-debian12@sha256:d75cdd72874d4790092fcb1b058493ecf6bb5bf2b2b897045b00ff01d91843f2

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]
