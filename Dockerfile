FROM golang:1.22 as builder

WORKDIR /app
COPY . /app

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -v -o app .

FROM alpine:3.19

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]