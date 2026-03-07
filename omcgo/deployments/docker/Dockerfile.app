FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/omcgo-app ./cmd/app

# ---

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /build/bin/omcgo-app /usr/local/bin/omcgo-app
COPY --from=builder /build/configs/app.yaml /etc/omcgo/app.yaml
COPY --from=builder /build/migrations /etc/omcgo/migrations

EXPOSE 8080 8443 9091 50051

ENTRYPOINT ["omcgo-app"]
CMD ["--config", "/etc/omcgo/app.yaml"]
