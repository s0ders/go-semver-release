FROM golang:1.22 as build

WORKDIR /build
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s" -o ./go-semver-release ./main.go

FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=build /build/go-semver-release ./go-semver-release

ENTRYPOINT ["./go-semver-release"]
CMD ["--help"]