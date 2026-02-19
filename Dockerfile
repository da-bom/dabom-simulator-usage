FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o simulator-usage ./cmd/simulator/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/simulator-usage /usr/local/bin/
COPY configs/config.yaml /etc/simulator-usage/config.yaml
EXPOSE 8080 9090
ENTRYPOINT ["simulator-usage"]
CMD ["--config", "/etc/simulator-usage/config.yaml"]
