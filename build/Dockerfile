FROM golang:1.19 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=1 go build ./cmd/go-semver-release/main.go

FROM gcr.io/distroless/base-debian11

COPY --from=build /go/src/app/main /go-semver-release

ENTRYPOINT ["/go-semver-release"]
CMD ["--help"]