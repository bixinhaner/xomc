FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-app ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-migrate ./cmd/migrate

# ---

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create log directory with proper permissions
RUN mkdir -p /var/log/omcgo && chmod 777 /var/log/omcgo

COPY --from=builder /build/bin/omcgo-app /usr/local/bin/omcgo-app
COPY --from=builder /build/bin/omcgo-migrate /usr/local/bin/omcgo-migrate
COPY --from=builder /build/cmd/app/etc/config.dev.yaml /etc/omcgo/app.yaml
COPY --from=builder /build/migrations /etc/omcgo/migrations
COPY deployments/docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8081 8444 9091 50051

ENTRYPOINT ["/entrypoint.sh"]
CMD ["--config", "/etc/omcgo/app.yaml"]
