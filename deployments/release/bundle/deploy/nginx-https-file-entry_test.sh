#!/usr/bin/env bash
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
RELEASE_DEPLOY="$REPO_ROOT/deployments/release/bundle/deploy"
NGINX_DEFAULT="$REPO_ROOT/deployments/docker/default.conf"
ENTRYPOINT="$REPO_ROOT/deployments/docker/docker-entrypoint.d/10-enable-https-file-entry.sh"
DEV_COMPOSE="$REPO_ROOT/deployments/docker/docker-compose.yml"
WEB_DOCKERFILE="$REPO_ROOT/deployments/docker/Dockerfile.web"
RELEASE_WEB_COMPOSE="$RELEASE_DEPLOY/docker-compose.web.yml"
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
expect_success "both certificate files absent keeps HTTPS disabled without breaking HTTP" env OMC_NGINX_HTTPS_CERT="$missing_cert" OMC_NGINX_HTTPS_KEY="$missing_key" OMC_NGINX_HTTPS_CONF="$out_conf" sh "$ENTRYPOINT"
[ ! -e "$out_conf" ] && ok || bad "disabled HTTPS entrypoint must not generate 8443 config"

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
	else
	  echo "WARN: openssl unavailable; certificate mismatch execution tests skipped" >&2
	fi

echo "-- install and healthcheck guards --"
valid_bash "install script" "$INSTALL"
valid_bash "healthcheck script" "$HEALTHCHECK"
valid_bash "real file-entry smoke script" "$SMOKE"
contains "install prechecks missing certificate" "Missing nginx HTTPS file-entry certificate" "$INSTALL"
contains "install prechecks missing private key" "Missing nginx HTTPS file-entry private key" "$INSTALL"
contains "install checks certificate/key match" "nginx HTTPS 文件入口证书与私钥不匹配" "$INSTALL"
contains "install keeps HTTP when HTTPS certificate is absent" "HTTP 8080 文件入口继续可用" "$INSTALL"
contains "healthcheck verifies nginx 8443 config" "web_https_file_entry_loaded" "$HEALTHCHECK"
contains "healthcheck requires every nginx 8443 config token" "grep -Fq 'listen 8443 ssl;' &&" "$HEALTHCHECK"
contains "healthcheck accepts disabled HTTPS entry" "web_https_file_entry_disabled" "$HEALTHCHECK"
contains "healthcheck tolerates nginx formatting whitespace" "tr -s '[:space:]' ' '" "$HEALTHCHECK"
contains "healthcheck verifies upload TLS reaches ACS" "https_file_entry_status_is /smallcell/FileUploadService 405" "$HEALTHCHECK"
contains "healthcheck verifies download TLS reaches ACS" "https_file_entry_status_is /smallcell/FileDownloadService/__healthcheck__/missing 404" "$HEALTHCHECK"
contains "healthcheck supports explicit real smoke" "--file-entry-smoke" "$HEALTHCHECK"

echo "-- real upload/download smoke contract --"
contains "smoke posts real HTTP upload body" "--data-binary" "$SMOKE"
contains "smoke posts HTTPS upload body" 'upload_file "$HTTPS_BASE"' "$SMOKE"
contains "smoke downloads through HTTPS 8443" "HTTPS 8443 downloads the HTTPS-uploaded object" "$SMOKE"
contains "smoke downloads through HTTP 8080" "HTTP 8080 downloads the HTTP-uploaded object" "$SMOKE"
contains "smoke compares downloaded content" "cmp -s" "$SMOKE"
contains "smoke uses ACS download bucket path" "/smallcell/FileDownloadService/logs/" "$SMOKE"

echo "-- release README operator contract --"
contains "README documents certificate path" "/etc/nginx/cert/cert.pem" "$README"
contains "README documents private-key path" "/etc/nginx/cert/key.pem" "$README"
contains "README documents absent certificate behavior" 'If both files are absent, `:8443` stays disabled' "$README"
contains "README documents HTTP compatibility when HTTPS disabled" 'available. If only one file exists' "$README"
contains "README documents HTTPS upload URL" "https://<OMC_PUBLIC_HOST>:8443/smallcell/FileUploadService" "$README"
contains "README documents HTTPS download URL" "https://<OMC_PUBLIC_HOST>:8443/smallcell/FileDownloadService" "$README"
contains "README states HTTP compatibility" 'HTTP `:8080` remains available' "$README"
contains "README documents smoke command" "smoke-nginx-https-file-entry.sh" "$README"

echo "Results: PASS=$PASS FAIL=$FAIL"
[ "$FAIL" -eq 0 ]
