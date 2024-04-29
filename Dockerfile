FROM golang:1.22 as builder

ARG APP_VERSION="v0.0.0+unknown"

WORKDIR /app
COPY . /app

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-X github.com/s0ders/go-semver-release/v2/cmd.cliVersion=$APP_VERSION -w -s" -v -o app .

FROM alpine:3.19

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]