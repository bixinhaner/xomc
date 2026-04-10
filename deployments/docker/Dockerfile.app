FROM golang:1.25-alpine AS builder

# 配置 DNS 服务器 (解决构建时 DNS 解析问题)
RUN echo "nameserver 223.5.5.5" > /etc/resolv.conf && \
    echo "nameserver 223.6.6.6" >> /etc/resolv.conf && \
    echo "nameserver 114.114.114.114" >> /etc/resolv.conf && \
    echo "nameserver 8.8.8.8" >> /etc/resolv.conf

# 安装 git (添加重试逻辑)
RUN for i in 1 2 3; do \
      apk add --no-cache git && break || \
      { echo "Attempt $i failed, retrying in 2s..."; sleep 2; }; \
    done

ARG GOPROXY=https://mirrors.aliyun.com/goproxy/,https://goproxy.cn,https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /build

COPY omcgo/go.mod omcgo/go.sum ./
RUN go mod download

COPY omcgo/ .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-app ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-migrate ./cmd/migrate

# ---

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create log directory with proper permissions
RUN mkdir -p /var/log/omcgo && chmod 777 /var/log/omcgo

COPY --from=builder /build/bin/omcgo-app /usr/local/bin/omcgo-app
COPY --from=builder /build/bin/omcgo-migrate /usr/local/bin/omcgo-migrate
COPY --from=builder /build/cmd/app/etc/config.dev.yaml /etc/omcgo/app.dev.yaml
COPY --from=builder /build/cmd/app/etc/config.test.yaml /etc/omcgo/app.test.yaml
COPY --from=builder /build/cmd/app/etc/config.prod.yaml /etc/omcgo/app.prod.yaml
COPY --from=builder /build/migrations /etc/omcgo/migrations
COPY deployments/docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV TZ=Asia/Shanghai
ENV OMCGO_SERVICE=app

EXPOSE 8081 8444 9091 50051

ENTRYPOINT ["/entrypoint.sh"]
