#!/bin/bash
# T-0118: 生成自签 TLS 证书，供 nginx web 容器终止 TLS 用
#
# 用法:
#   bash deployments/docker/scripts/generate-self-signed-tls.sh [host_ip] [days]
#
# 参数:
#   host_ip — 部署 host IP，默认 172.19.1.73；会写入证书 SAN
#   days   — 证书有效期天数，默认 1825（5 年）
#
# 输出:
#   deployments/docker/certs/server.crt
#   deployments/docker/certs/server.key
#   deployments/docker/certs/openssl.cnf
#
# 与之配套的下一步参见 TLS-SETUP.md。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CERT_DIR="$SCRIPT_DIR/../certs"
mkdir -p "$CERT_DIR"

HOST_IP="${1:-172.19.1.73}"
DAYS="${2:-1825}"

if ! command -v openssl >/dev/null 2>&1; then
    echo "❌ openssl 未安装；请先 apt/brew install openssl" >&2
    exit 1
fi

cat > "$CERT_DIR/openssl.cnf" <<EOF
[req]
default_bits       = 2048
prompt             = no
default_md         = sha256
distinguished_name = dn
req_extensions     = req_ext

[dn]
C  = CN
O  = OMC
CN = ${HOST_IP}

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1  = 127.0.0.1
IP.2  = ${HOST_IP}
EOF

openssl req -x509 -nodes -newkey rsa:2048 \
    -keyout "$CERT_DIR/server.key" \
    -out    "$CERT_DIR/server.crt" \
    -days   "$DAYS" \
    -config "$CERT_DIR/openssl.cnf" \
    -extensions req_ext \
    2>/dev/null

chmod 644 "$CERT_DIR/server.crt"
chmod 600 "$CERT_DIR/server.key"

echo "✅ 自签证书已生成"
echo "   cert: $CERT_DIR/server.crt"
echo "   key:  $CERT_DIR/server.key"
echo "   SAN:  localhost, 127.0.0.1, $HOST_IP"
echo "   过期: $DAYS 天后"
echo ""
echo "下一步（参 deployments/docker/TLS-SETUP.md）："
echo "  1. cp deployments/docker/default-tls.conf.example deployments/docker/default-tls.conf"
echo "  2. 修改 deployments/docker/docker-compose.yml web 服务："
echo "       - 加端口映射 '443:443'"
echo "       - volumes 加 './certs:/etc/nginx/certs:ro' 和"
echo "                    './default-tls.conf:/etc/nginx/conf.d/default-tls.conf:ro'"
echo "  3. cd deployments/docker && docker-compose up -d web"
echo "  4. 浏览器访问 https://${HOST_IP}（首次需信任自签证书 → 高级 → 继续访问）"
