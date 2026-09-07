FROM golang:1.24 AS builder
WORKDIR /src

COPY apps/app-backend/ /src/apps/app-backend/
COPY db/ /src/db/
WORKDIR /src/apps/app-backend
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /out/app /app/app
COPY --from=builder /src/db /db
EXPOSE 8080
ENTRYPOINT ["/app/app"]
