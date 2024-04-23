FROM golang:1.22 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=1 go build -ldflags "-w -s" -o ./main ./main.go

FROM alpine:3.19

COPY --from=build /go/src/app/main /go-semver-release

ENTRYPOINT ["/go-semver-release"]
CMD ["--help"]