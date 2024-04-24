FROM golang:1.22 as build

WORKDIR /build
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags "-w -s" -o ./go-semver-release ./main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=build /build/go-semver-release .

ENTRYPOINT ["./go-semver-release"]
CMD ["--help"]