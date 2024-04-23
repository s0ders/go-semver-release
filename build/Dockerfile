FROM golang:1.22 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build ./cmd/go-semver-release/main.go

FROM gcr.io/distroless/base-nossl-debian12

COPY --from=build /go/src/app/main /go-semver-release

ENTRYPOINT ["/go-semver-release"]
CMD ["--help"]