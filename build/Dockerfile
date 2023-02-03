# Start by building the application.
FROM golang:1.19 as build

WORKDIR /go/src/app
COPY go.* .

RUN go mod download

COPY . .
RUN go build -o /go/bin/go-semver-release

# Now copy it into our base image.
FROM gcr.io/distroless/base-debian11
COPY --from=build /go/bin/go-semver-release /
ENTRYPOINT ["/go-semver-release"]
CMD ["--help"]