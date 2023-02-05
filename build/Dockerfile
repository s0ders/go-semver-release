# Start by building the application.
FROM golang:1.19 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN go build ./cmd/go-semver-release -o /go/bin/app

# Now copy it into our base image.
FROM gcr.io/distroless/base-debian11
COPY --from=build /go/bin/app /
RUN chmod +x /app
ENTRYPOINT ["/app"]
CMD ["--help"]