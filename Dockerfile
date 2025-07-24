FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o blockwatch cmd/cli/main.go

FROM scratch

COPY --from=builder /app/blockwatch /blockwatch
ENTRYPOINT ["/blockwatch"]
