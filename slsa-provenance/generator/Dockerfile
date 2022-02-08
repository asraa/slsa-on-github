# golang:1.17.1-alpine (linux/amd64)
FROM golang@sha256:13919fb9091f6667cb375d5fdf016ecd6d3a5d5995603000d422b04583de4ef9 AS builder

WORKDIR /app
COPY . /app

RUN go mod vendor
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -v -o app ./cmd

FROM gcr.io/distroless/static@sha256:8ad6f3ec70dad966479b9fb48da991138c72ba969859098ec689d1450c2e6c97

COPY --from=builder /app/app /app

ENTRYPOINT ["/app"]
