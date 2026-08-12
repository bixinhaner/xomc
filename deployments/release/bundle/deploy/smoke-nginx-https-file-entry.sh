#!/usr/bin/env bash
# Verify the nginx 8443 HTTPS file entry with real upload/download traffic.
#
# Defaults target the local Docker stack. Override these when validating a
# release host:
#   OMC_FILE_HTTPS_BASE=https://<host>:8443
#   OMC_FILE_HTTP_BASE=http://<host>:8080
set -euo pipefail

HTTPS_BASE="${OMC_FILE_HTTPS_BASE:-https://127.0.0.1:8443}"
HTTP_BASE="${OMC_FILE_HTTP_BASE:-http://127.0.0.1:8080}"
CURL_TLS_ARGS=()
if [ "${OMC_FILE_HTTPS_INSECURE:-1}" = "1" ]; then
  CURL_TLS_ARGS=(-k)
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

STAMP="$(date +%Y%m%d%H%M%S)"
TASK_ID="301${STAMP}abcdef01"
SN="XOMC301SMOKE"

http_payload="$TMP_DIR/http-upload.txt"
https_payload="$TMP_DIR/https-upload.txt"
printf 'xomc-301-http-download-ok\n' >"$http_payload"
printf 'xomc-301-https-download-ok\n' >"$https_payload"

json_value() { # json_value <key>
  sed -n 's/.*"'"$1"'":"\([^"]*\)".*/\1/p'
}

upload_file() { # upload_file <base> <payload_path> <filename> [curl_tls_args...]
  local base="$1" payload="$2" filename="$3"
  shift 3
  curl "$@" -fsS --max-time 10 \
    -X POST \
    --data-binary "@${payload}" \
    "${base}/smallcell/FileUploadService?fileType=LOG&sn=${SN}&taskId=${TASK_ID}&filename=${filename}"
}

download_file() { # download_file <base> <object_path> <output_path> [curl_tls_args...]
  local base="$1" object_path="$2" output="$3"
  shift 3
  curl "$@" -fsS --max-time 10 \
    "${base}/smallcell/FileDownloadService/logs/${object_path}" \
    -o "$output"
}

assert_same_file() { # assert_same_file <expected> <actual> <label>
  local expected="$1" actual="$2" label="$3"
  if cmp -s "$expected" "$actual"; then
    printf '[OK]   %s\n' "$label"
  else
    printf '[FAIL] %s\n' "$label" >&2
    printf 'expected:\n' >&2
    cat "$expected" >&2
    printf '\nactual:\n' >&2
    cat "$actual" >&2
    return 1
  fi
}

http_json="$(upload_file "$HTTP_BASE" "$http_payload" "xomc-301-http-${STAMP}.txt")"
https_json="$(upload_file "$HTTPS_BASE" "$https_payload" "xomc-301-https-${STAMP}.txt" "${CURL_TLS_ARGS[@]}")"

http_path="$(printf '%s\n' "$http_json" | json_value path)"
https_path="$(printf '%s\n' "$https_json" | json_value path)"
[ -n "$http_path" ] || { echo "[FAIL] HTTP upload response did not include path: $http_json" >&2; exit 1; }
[ -n "$https_path" ] || { echo "[FAIL] HTTPS upload response did not include path: $https_json" >&2; exit 1; }
printf '[OK]   HTTP upload stored logs/%s\n' "$http_path"
printf '[OK]   HTTPS upload stored logs/%s\n' "$https_path"

download_file "$HTTP_BASE" "$http_path" "$TMP_DIR/http-via-http.txt"
assert_same_file "$http_payload" "$TMP_DIR/http-via-http.txt" "HTTP 8080 downloads the HTTP-uploaded object"

download_file "$HTTPS_BASE" "$https_path" "$TMP_DIR/https-via-https.txt" "${CURL_TLS_ARGS[@]}"
assert_same_file "$https_payload" "$TMP_DIR/https-via-https.txt" "HTTPS 8443 downloads the HTTPS-uploaded object"

download_file "$HTTPS_BASE" "$http_path" "$TMP_DIR/http-via-https.txt" "${CURL_TLS_ARGS[@]}"
assert_same_file "$http_payload" "$TMP_DIR/http-via-https.txt" "HTTPS 8443 downloads an HTTP-uploaded object"

download_file "$HTTP_BASE" "$https_path" "$TMP_DIR/https-via-http.txt"
assert_same_file "$https_payload" "$TMP_DIR/https-via-http.txt" "HTTP 8080 downloads an HTTPS-uploaded object"

echo "PASS: nginx HTTPS file entry upload/download smoke passed"
