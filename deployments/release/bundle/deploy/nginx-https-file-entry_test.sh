#!/usr/bin/env bash
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
RELEASE_DEPLOY="$REPO_ROOT/deployments/release/bundle/deploy"
BUILD_RELEASE="$REPO_ROOT/deployments/release/build-release.sh"
GITIGNORE="$REPO_ROOT/.gitignore"
REPO_CERT_DIR="$RELEASE_DEPLOY/nginx-cert"
NGINX_DEFAULT="$REPO_ROOT/deployments/docker/default.conf"
ENTRYPOINT="$REPO_ROOT/deployments/docker/docker-entrypoint.d/10-enable-https-file-entry.sh"
DEV_COMPOSE="$REPO_ROOT/deployments/docker/docker-compose.yml"
WEB_DOCKERFILE="$REPO_ROOT/deployments/docker/Dockerfile.web"
RELEASE_WEB_COMPOSE="$RELEASE_DEPLOY/docker-compose.web.yml"
OPENSSL_LEGACY_CONF="$RELEASE_DEPLOY/openssl-legacy.cnf"
INSTALL="$RELEASE_DEPLOY/install.sh"
HEALTHCHECK="$RELEASE_DEPLOY/healthcheck.sh"
SMOKE="$RELEASE_DEPLOY/smoke-nginx-https-file-entry.sh"
README="$RELEASE_DEPLOY/README.md"

PASS=0
FAIL=0
fixture_pid=""
ok() { PASS=$((PASS + 1)); }
bad() { echo "FAIL: $*" >&2; FAIL=$((FAIL + 1)); }
contains() {
  local name="$1" pattern="$2" file="$3"
  if grep -Fq -- "$pattern" "$file"; then ok; else bad "$name: $file missing [$pattern]"; fi
}
not_contains() {
  local name="$1" pattern="$2" file="$3"
  if grep -Fq -- "$pattern" "$file"; then bad "$name: $file must not contain [$pattern]"; else ok; fi
}
appears_before() {
  local name="$1" first="$2" second="$3" file="$4"
  local first_line second_line
  first_line="$(grep -nF -- "$first" "$file" | head -1 | cut -d: -f1)"
  second_line="$(grep -nF -- "$second" "$file" | head -1 | cut -d: -f1)"
  if [ -n "$first_line" ] && [ -n "$second_line" ] && [ "$first_line" -lt "$second_line" ]; then
    ok
  else
    bad "$name: [$first] must appear before [$second]"
  fi
}
valid_bash() {
  local name="$1" file="$2"
  if bash -n "$file"; then ok; else bad "$name: $file has Bash syntax errors"; fi
}
expect_success() {
  local name="$1"; shift
  local output
  output="$(mktemp)"
  if "$@" >"$output" 2>&1; then
    ok
  else
    cat "$output" >&2
    bad "$name: command failed unexpectedly"
  fi
  rm -f "$output"
}
expect_fail() {
  local name="$1"; shift
  if "$@" >/dev/null 2>&1; then bad "$name: command succeeded unexpectedly"; else ok; fi
}

echo "-- nginx 8443 HTTPS file entry --"
valid_bash "nginx HTTPS file-entry entrypoint" "$ENTRYPOINT"
not_contains "default nginx config must not require certificates to start HTTP" "listen 8443 ssl;" "$NGINX_DEFAULT"
contains "entrypoint generates nginx 8443 TLS server" "listen 8443 ssl;" "$ENTRYPOINT"
contains "entrypoint loads deployment certificate" 'ssl_certificate     $CERT;' "$ENTRYPOINT"
contains "entrypoint loads deployment private key" 'ssl_certificate_key $KEY;' "$ENTRYPOINT"
contains "entrypoint forwards HTTPS upload/download to ACS HTTP service" "proxy_pass http://acs_backend;" "$ENTRYPOINT"
contains "entrypoint marks HTTPS upstream scheme" "proxy_set_header X-Forwarded-Proto https;" "$ENTRYPOINT"
contains "nginx keeps HTTP ACS entry" "listen 8080;" "$NGINX_DEFAULT"
contains "nginx keeps upload/download body streaming" "proxy_request_buffering off;" "$NGINX_DEFAULT"
not_contains "repository must not contain certificate material" "BEGIN PRIVATE KEY" "$NGINX_DEFAULT"
not_contains "repository must not contain certificate material" "BEGIN CERTIFICATE" "$NGINX_DEFAULT"

echo "-- compose ports and certificate mount --"
contains "development compose publishes 8443" '- "8443:8443"' "$DEV_COMPOSE"
contains "development compose mounts deployment certificate directory" "- /etc/nginx/cert:/etc/nginx/cert:ro" "$DEV_COMPOSE"
contains "web image documents 8443" "EXPOSE 8080 8081 8443" "$WEB_DOCKERFILE"
contains "web image installs HTTPS file-entry entrypoint" "10-enable-https-file-entry.sh" "$WEB_DOCKERFILE"
contains "release compose publishes 8443" '- "8443:8443"' "$RELEASE_WEB_COMPOSE"
contains "release compose mounts deployment certificate directory" "- /etc/nginx/cert:/etc/nginx/cert:ro" "$RELEASE_WEB_COMPOSE"
contains "release compose sets web-only OpenSSL legacy config" "OPENSSL_CONF: /etc/nginx/openssl-legacy.cnf" "$RELEASE_WEB_COMPOSE"
contains "release compose mounts OpenSSL legacy config read-only" "- ./openssl-legacy.cnf:/etc/nginx/openssl-legacy.cnf:ro" "$RELEASE_WEB_COMPOSE"
contains "OpenSSL legacy config exists" "openssl_conf = default_conf" "$OPENSSL_LEGACY_CONF"
contains "OpenSSL legacy config lowers security level for old 1024-bit certificate" "CipherString = DEFAULT:@SECLEVEL=0" "$OPENSSL_LEGACY_CONF"

echo "-- executable certificate failure paths --"
TMP="$(mktemp -d)"
cleanup() {
  if [ -n "${fixture_pid:-}" ]; then
    kill "$fixture_pid" >/dev/null 2>&1 || true
    wait "$fixture_pid" >/dev/null 2>&1 || true
  fi
  rm -rf "$TMP"
}
trap cleanup EXIT
missing_cert="$TMP/missing-cert.pem"
missing_key="$TMP/missing-key.pem"
out_conf="$TMP/https-file-entry.conf"
expect_success "container entrypoint keeps local HTTP-only startup when both certificate files are absent" env OMC_NGINX_HTTPS_CERT="$missing_cert" OMC_NGINX_HTTPS_KEY="$missing_key" OMC_NGINX_HTTPS_CONF="$out_conf" sh "$ENTRYPOINT"
[ ! -e "$out_conf" ] && ok || bad "local HTTP-only entrypoint compatibility must not generate 8443 config"

if command -v openssl >/dev/null 2>&1; then
  cert_a="$TMP/cert-a.pem"; key_a="$TMP/key-a.pem"
  cert_b="$TMP/cert-b.pem"; key_b="$TMP/key-b.pem"
  openssl req -x509 -nodes -newkey rsa:2048 -keyout "$key_a" -out "$cert_a" -days 1 -subj /CN=xomc-a >/dev/null 2>&1
  openssl req -x509 -nodes -newkey rsa:2048 -keyout "$key_b" -out "$cert_b" -days 1 -subj /CN=xomc-b >/dev/null 2>&1
  expect_fail "partial certificate pair fails startup" env OMC_NGINX_HTTPS_CERT="$cert_a" OMC_NGINX_HTTPS_KEY="$missing_key" OMC_NGINX_HTTPS_CONF="$out_conf" sh "$ENTRYPOINT"
  expect_fail "mismatched certificate and key fail startup" env OMC_NGINX_HTTPS_CERT="$cert_a" OMC_NGINX_HTTPS_KEY="$key_b" OMC_NGINX_HTTPS_CONF="$out_conf" sh "$ENTRYPOINT"
  printf 'not a certificate\n' >"$TMP/bad-cert.pem"
  expect_fail "unparseable certificate fails startup" env OMC_NGINX_HTTPS_CERT="$TMP/bad-cert.pem" OMC_NGINX_HTTPS_KEY="$key_a" OMC_NGINX_HTTPS_CONF="$out_conf" sh "$ENTRYPOINT"
  expect_success "valid certificate pair generates HTTPS config" env OMC_NGINX_HTTPS_CERT="$cert_a" OMC_NGINX_HTTPS_KEY="$key_a" OMC_NGINX_HTTPS_CONF="$out_conf" sh "$ENTRYPOINT"
  contains "generated config listens on 8443" "listen 8443 ssl;" "$out_conf"
  if command -v python3 >/dev/null 2>&1; then
    fixture_py="$TMP/file-entry-fixture.py"
    ports_file="$TMP/file-entry-ports"
    cat >"$fixture_py" <<'PY'
import http.server
import json
import ssl
import sys
import threading
import urllib.parse

cert_file, key_file, ports_file = sys.argv[1:4]
objects = {}

class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def do_POST(self):
        parsed = urllib.parse.urlparse(self.path)
        if parsed.path != "/smallcell/FileUploadService":
            self.send_error(404)
            return
        params = urllib.parse.parse_qs(parsed.query)
        filename = params.get("filename", ["upload.bin"])[0]
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length)
        object_path = "running/test/" + filename
        objects[object_path] = body
        payload = json.dumps({"path": object_path}, separators=(",", ":")).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def do_GET(self):
        prefix = "/smallcell/FileDownloadService/logs/"
        parsed = urllib.parse.urlparse(self.path)
        if not parsed.path.startswith(prefix):
            self.send_error(404)
            return
        object_path = urllib.parse.unquote(parsed.path[len(prefix):])
        body = objects.get(object_path)
        if body is None:
            self.send_error(404)
            return
        self.send_response(200)
        self.send_header("Content-Type", "application/octet-stream")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *_args):
        return

class Server(http.server.ThreadingHTTPServer):
    daemon_threads = True

httpd = Server(("127.0.0.1", 0), Handler)
httpsd = Server(("127.0.0.1", 0), Handler)
context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
context.load_cert_chain(cert_file, key_file)
httpsd.socket = context.wrap_socket(httpsd.socket, server_side=True)

with open(ports_file, "w", encoding="utf-8") as fh:
    fh.write(f"{httpd.server_port} {httpsd.server_port}\n")

threading.Thread(target=httpd.serve_forever, daemon=True).start()
httpsd.serve_forever()
PY
    fixture_log="$TMP/file-entry-fixture.log"
    python3 "$fixture_py" "$cert_a" "$key_a" "$ports_file" >"$fixture_log" 2>&1 &
    fixture_pid="$!"
    for _ in 1 2 3 4 5 6 7 8 9 10; do
      [ -s "$ports_file" ] && break
      sleep 0.2
    done
    if [ -s "$ports_file" ]; then
      read -r http_port https_port <"$ports_file"
      expect_success "real upload/download smoke works with temporary HTTPS certificate" env OMC_FILE_HTTP_BASE="http://127.0.0.1:${http_port}" OMC_FILE_HTTPS_BASE="https://127.0.0.1:${https_port}" OMC_FILE_HTTPS_INSECURE=1 bash "$SMOKE"
    else
      [ ! -s "$fixture_log" ] || cat "$fixture_log" >&2
      bad "temporary HTTP/HTTPS upload/download fixture did not start"
    fi
  else
    bad "python3 is required for temporary-certificate upload/download smoke fixture"
  fi
  if command -v docker >/dev/null 2>&1 && docker image inspect nginx:alpine >/dev/null 2>&1; then
    nginx_legacy_conf="$TMP/nginx-1024-test.conf"
    cat >"$nginx_legacy_conf" <<'NGINX'
events {}
http {
  server {
    listen 8443 ssl;
    ssl_certificate /etc/nginx/cert/cert.pem;
    ssl_certificate_key /etc/nginx/cert/key.pem;
    location / {
      return 200 "ok\n";
    }
  }
}
NGINX
    expect_success "nginx config test accepts repository 1024-bit certificate with web-only OpenSSL legacy config" \
      docker run --network none --rm \
        -e OPENSSL_CONF=/etc/nginx/openssl-legacy.cnf \
        -v "$OPENSSL_LEGACY_CONF:/etc/nginx/openssl-legacy.cnf:ro" \
        -v "$REPO_CERT_DIR:/etc/nginx/cert:ro" \
        -v "$nginx_legacy_conf:/etc/nginx/nginx.conf:ro" \
        nginx:alpine nginx -t
  else
    echo "WARN: docker or local nginx:alpine image unavailable; 1024-bit nginx config execution test skipped" >&2
  fi
else
  echo "WARN: openssl unavailable; certificate mismatch execution tests skipped" >&2
fi

echo "-- install and healthcheck guards --"
valid_bash "build release script" "$BUILD_RELEASE"
valid_bash "install script" "$INSTALL"
valid_bash "healthcheck script" "$HEALTHCHECK"
valid_bash "real file-entry smoke script" "$SMOKE"
contains "release build uses repository certificate path" "deployments/release/bundle/deploy/nginx-cert" "$BUILD_RELEASE"
contains "release build copies deploy bundle including OpenSSL legacy config" 'cp -r "$SCRIPT_DIR/bundle/deploy"' "$BUILD_RELEASE"
not_contains "release build must not depend on private certificate override" "OMC_RELEASE_HTTPS_CERT_SOURCE_DIR" "$BUILD_RELEASE"
contains "release build copies certificate assets into package" "copy_release_https_cert_assets" "$BUILD_RELEASE"
contains "release build packages fixed certificate path" "deploy/nginx-cert" "$BUILD_RELEASE"
not_contains "gitignore must not block repository release certificate assets" "deployments/release/private/" "$GITIGNORE"
contains "repository carries old OMC certificate" "BEGIN CERTIFICATE" "$REPO_CERT_DIR/cert.pem"
contains "repository carries old OMC private key" "BEGIN PRIVATE KEY" "$REPO_CERT_DIR/key.pem"
expect_success "release build validates repository certificate assets by default" bash "$BUILD_RELEASE" --verify-https-cert-assets-only
expect_success "release build verifies final package certificate layout" bash "$BUILD_RELEASE" --verify-https-cert-package-layout-only
if command -v git >/dev/null 2>&1; then
  expect_fail "repository certificate must not be ignored by git" git -C "$REPO_ROOT" check-ignore deployments/release/bundle/deploy/nginx-cert/cert.pem
  expect_fail "repository private key must not be ignored by git" git -C "$REPO_ROOT" check-ignore deployments/release/bundle/deploy/nginx-cert/key.pem
  expect_success "repository certificate is tracked by git" git -C "$REPO_ROOT" ls-files --error-unmatch deployments/release/bundle/deploy/nginx-cert/cert.pem
  expect_success "repository private key is tracked by git" git -C "$REPO_ROOT" ls-files --error-unmatch deployments/release/bundle/deploy/nginx-cert/key.pem
else
  bad "git is required to verify repository certificate ignore rules"
fi
contains "install prechecks packaged certificate" "Packaged nginx HTTPS file-entry certificate precheck passed" "$INSTALL"
contains "install installs packaged certificate to host" "install_nginx_https_cert" "$INSTALL"
appears_before "install installs HTTPS certificate before web container startup" "install_nginx_https_cert" '"${DC[@]}" up --pull never -d' "$INSTALL"
contains "install writes host certificate path" "/etc/nginx/cert" "$INSTALL"
contains "install writes host private key with restrictive mode" "install -m 0600" "$INSTALL"
contains "install checks certificate/key match" "证书与私钥不匹配" "$INSTALL"
not_contains "install must not allow missing certificate as release success" "8443 将不启用" "$INSTALL"
contains "healthcheck verifies nginx 8443 config" "web_https_file_entry_loaded" "$HEALTHCHECK"
contains "healthcheck requires every nginx 8443 config token" "grep -Fq 'listen 8443 ssl;' &&" "$HEALTHCHECK"
contains "healthcheck requires installed HTTPS certificate" "web HTTPS file-entry certificate installed" "$HEALTHCHECK"
contains "healthcheck tolerates nginx formatting whitespace" "tr -s '[:space:]' ' '" "$HEALTHCHECK"
contains "healthcheck verifies HTTPS ACS healthz" "https_file_entry_status_is /healthz 200" "$HEALTHCHECK"
contains "healthcheck verifies HTTPS ACS service path" "https_file_entry_status_is /smallcell/AcsService 405" "$HEALTHCHECK"
contains "healthcheck verifies upload TLS reaches ACS" "https_file_entry_status_is /smallcell/FileUploadService 405" "$HEALTHCHECK"
contains "healthcheck verifies download TLS reaches ACS" "https_file_entry_status_is /smallcell/FileDownloadService/health-check/missing 404" "$HEALTHCHECK"
contains "healthcheck supports explicit real smoke" "--file-entry-smoke" "$HEALTHCHECK"

echo "-- real upload/download smoke contract --"
contains "smoke posts real HTTP upload body" "--data-binary" "$SMOKE"
contains "smoke posts HTTPS upload body" 'upload_file "$HTTPS_BASE"' "$SMOKE"
contains "smoke downloads through HTTPS 8443" "HTTPS 8443 downloads the HTTPS-uploaded object" "$SMOKE"
contains "smoke downloads through HTTP 8080" "HTTP 8080 downloads the HTTP-uploaded object" "$SMOKE"
contains "smoke compares downloaded content" "cmp -s" "$SMOKE"
contains "smoke uses ACS download bucket path" "/smallcell/FileDownloadService/logs/" "$SMOKE"

echo "-- release README operator contract --"
contains "README documents packaged certificate path" "deploy/nginx-cert/cert.pem" "$README"
contains "README documents repository certificate path" "deployments/release/bundle/deploy/nginx-cert/cert.pem" "$README"
contains "README documents old OMC certificate source" "root@172.21.175.129:/etc/nginx/cert/cert.pem" "$README"
contains "README says release build does not depend on local private certificate directory" "depend on any local private certificate directory" "$README"
contains "README documents certificate path" "/etc/nginx/cert/cert.pem" "$README"
contains "README documents private-key path" "/etc/nginx/cert/key.pem" "$README"
contains "README documents install-time certificate copy" "installs it to" "$README"
contains "README documents missing certificate as failure" "fails before the web container starts" "$README"
contains "README documents skip-web is not valid for HTTPS 8443 acceptance" "Do not use" "$README"
contains "README documents HTTPS upload URL" "https://<OMC_PUBLIC_HOST>:8443/smallcell/FileUploadService" "$README"
contains "README documents HTTPS download URL" "https://<OMC_PUBLIC_HOST>:8443/smallcell/FileDownloadService" "$README"
contains "README documents HTTPS ACS service URL" "https://<OMC_PUBLIC_HOST>:8443/smallcell/AcsService" "$README"
contains "README documents web-only OpenSSL legacy config" "OPENSSL_CONF=/etc/nginx/openssl-legacy.cnf" "$README"
contains "README documents legacy certificate security exception" "1024-bit" "$README"
contains "README documents removing legacy exception after certificate replacement" "remove this exception" "$README"
contains "README documents real base-station final acceptance" "real base station Inform" "$README"
contains "README states HTTP compatibility" 'HTTP `:8080` remains available' "$README"
contains "README documents 8443 healthz check" "https://127.0.0.1:8443/healthz" "$README"
contains "README documents smoke command" "smoke-nginx-https-file-entry.sh" "$README"

echo "Results: PASS=$PASS FAIL=$FAIL"
[ "$FAIL" -eq 0 ]
