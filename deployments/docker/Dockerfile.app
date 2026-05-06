# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder

# 国内环境加速 apk 源（dl-cdn.alpinelinux.org 国内常被墙），可通过 --build-arg
# APK_MIRROR=dl-cdn.alpinelinux.org 切回官方源。
ARG APK_MIRROR=mirrors.aliyun.com
RUN sed -i "s|dl-cdn.alpinelinux.org|${APK_MIRROR}|g" /etc/apk/repositories

RUN apk add --no-cache git

ARG GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /build

COPY omcgo/ .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-app ./cmd/app && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-migrate ./cmd/migrate

# ---

FROM alpine:3.19

ARG APK_MIRROR=mirrors.aliyun.com
RUN sed -i "s|dl-cdn.alpinelinux.org|${APK_MIRROR}|g" /etc/apk/repositories


RUN apk add --no-cache ca-certificates tzdata

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
