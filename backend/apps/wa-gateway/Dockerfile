FROM golang:1.25 AS builder
WORKDIR /src

COPY apps/wa-gateway/ /src/apps/wa-gateway/
WORKDIR /src/apps/wa-gateway
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/wa-gateway ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /out/wa-gateway /app/wa-gateway
EXPOSE 8090
ENTRYPOINT ["/app/wa-gateway"]
