# Start by building the application.
FROM golang:1.20 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=1 go build ./cmd/main.go -o /go/bin/go-semver-release

# Now copy it into our base image.
FROM gcr.io/distroless/base-debian11

COPY --from=build /go/bin/go-semver-release /

ENTRYPOINT ["/go-semver-release"]
CMD ["--help"]