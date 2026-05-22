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
# datamodels/：mml-catalog/ 启动期由 catalogloader 加载；templates/ 预留。
# 原 datamodels/seed/ 是 omcgo-seed JSON 种子链路，已下线；保留目录以兼容文档。
COPY --from=builder /build/datamodels /etc/omcgo/datamodels
# T-0098 dictloader 启动期加载的 4 域字典 XML（param-mappings / indicator-library /
# alarm-definitions / products）。config.dev.yaml 用相对路径 xml_base_dir: "data"，
# entrypoint.sh 把 cwd 切到 /etc/omcgo 让相对路径解析正确。
COPY --from=builder /build/data /etc/omcgo/data
# Casbin RBAC 模型文件：admin provider 启动期加载 configs/casbin_model.conf
# （相对路径，由 entrypoint.sh 切到 /etc/omcgo 后解析）。
# 缺失会导致 NewCasbinAuthorizer 失败 → roleRepo.authorizer == nil →
# 所有走 RequireAPIPermission/RequirePermission 的端点对非超管用户返
# 500 "casbin authorizer not configured"
# （pg_role_repository.go::CheckPermission）。
COPY --from=builder /build/configs /etc/omcgo/configs
COPY deployments/docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV TZ=Asia/Shanghai
ENV OMCGO_SERVICE=app

EXPOSE 8081 8444 9091 50051

ENTRYPOINT ["/entrypoint.sh"]
