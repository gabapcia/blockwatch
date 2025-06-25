FROM golang:1.24 AS builder

WORKDIR /app

COPY . .
RUN make build

FROM scratch

COPY --from=builder /app/blockwatch /blockwatch

ENTRYPOINT ["/blockwatch"]
