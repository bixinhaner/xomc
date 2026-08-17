#!/usr/bin/env bash
set -euo pipefail

# Configure one logical OMC's shared SMTP sender through the audited admin API.
# Secrets are accepted only through environment variables and are never echoed.

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "Missing required environment variable: ${name}" >&2
    exit 2
  fi
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Required command not found: $1" >&2
    exit 2
  fi
}

require_command curl
require_command jq
require_env OMC_URL
require_env ADMIN_TOKEN

SMTP_ENABLED="${SMTP_ENABLED:-true}"
SMTP_PORT="${SMTP_PORT:-587}"
SMTP_SECURITY_MODE="${SMTP_SECURITY_MODE:-starttls}"
SMTP_AUTH_ENABLED="${SMTP_AUTH_ENABLED:-true}"
SMTP_TIMEOUT_SECONDS="${SMTP_TIMEOUT_SECONDS:-10}"
SMTP_FROM_NAME="${SMTP_FROM_NAME:-OMC}"
KEEP_EXISTING_PASSWORD="${KEEP_EXISTING_PASSWORD:-false}"
SEND_TEST_EMAIL="${SEND_TEST_EMAIL:-false}"

if [[ "${SMTP_ENABLED}" == "true" ]]; then
  require_env SMTP_HOST
  require_env SMTP_FROM_ADDRESS
fi
if [[ "${SMTP_ENABLED}" == "true" && "${SMTP_AUTH_ENABLED}" == "true" ]]; then
  require_env SMTP_USERNAME
  if [[ -z "${SMTP_PASSWORD:-}" && "${KEEP_EXISTING_PASSWORD}" != "true" ]]; then
    echo "SMTP_PASSWORD is required for first-time authenticated SMTP configuration." >&2
    echo "Set KEEP_EXISTING_PASSWORD=true only when rotating non-password fields." >&2
    exit 2
  fi
fi
if [[ "${SEND_TEST_EMAIL}" == "true" ]]; then
  require_env TEST_RECIPIENT
fi

umask 077
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/omc-email-config.XXXXXX")"
items_file="${work_dir}/items.json"
password_file="${work_dir}/smtp-password"
payload_file="${work_dir}/payload.json"
response_file="${work_dir}/response.json"
curl_config="${work_dir}/curl.conf"
test_payload_file="${work_dir}/test-payload.json"
test_response_file="${work_dir}/test-response.json"

cleanup() {
  rm -f "${items_file}" "${password_file}" "${payload_file}" "${response_file}" \
    "${curl_config}" "${curl_config}.test" "${test_payload_file}" "${test_response_file}"
  rmdir "${work_dir}" 2>/dev/null || true
}
trap cleanup EXIT

jq -n \
    --arg enabled "${SMTP_ENABLED}" \
    --arg host "${SMTP_HOST:-}" \
    --arg port "${SMTP_PORT}" \
    --arg security "${SMTP_SECURITY_MODE}" \
    --arg auth "${SMTP_AUTH_ENABLED}" \
    --arg username "${SMTP_USERNAME:-}" \
    --arg fromAddress "${SMTP_FROM_ADDRESS:-}" \
    --arg fromName "${SMTP_FROM_NAME}" \
    --arg timeout "${SMTP_TIMEOUT_SECONDS}" \
    '[
      {key:"enabled", value:$enabled, value_type:"bool"},
      {key:"host", value:$host, value_type:"string"},
      {key:"port", value:$port, value_type:"int"},
      {key:"security_mode", value:$security, value_type:"string"},
      {key:"auth_enabled", value:$auth, value_type:"bool"},
      {key:"username", value:$username, value_type:"string"},
      {key:"from_address", value:$fromAddress, value_type:"string"},
      {key:"from_name", value:$fromName, value_type:"string"},
      {key:"timeout_seconds", value:$timeout, value_type:"int"}
    ]' >"${items_file}"

printf '%s' "${SMTP_PASSWORD:-}" >"${password_file}"
include_password=false
if [[ -n "${SMTP_PASSWORD:-}" ]]; then
  include_password=true
fi
jq -n \
  --slurpfile baseItems "${items_file}" \
  --rawfile password "${password_file}" \
  --argjson includePassword "${include_password}" \
  '{
    category:"notification.email",
    items: ($baseItems[0] + (if $includePassword then [{key:"password", value:$password, value_type:"string"}] else [] end))
  }' >"${payload_file}"

api_base="${OMC_URL%/}/api/v1"
{
  printf 'url = "%s"\n' "${api_base}/admin/sysConfig/batch"
  printf 'request = "POST"\n'
  printf 'header = "Authorization: Bearer %s"\n' "${ADMIN_TOKEN}"
  printf 'header = "Content-Type: application/json"\n'
  printf 'data-binary = "@%s"\n' "${payload_file}"
  printf 'output = "%s"\n' "${response_file}"
  printf 'fail-with-body\nsilent\nshow-error\n'
} >"${curl_config}"

# Do not propagate credentials to jq/curl child processes. Their argv only
# contains paths to owner-readable temporary files.
unset SMTP_PASSWORD ADMIN_TOKEN
curl --config "${curl_config}"

if [[ "$(jq -r '.ret // 0' "${response_file}")" != "0" ]]; then
  echo "SMTP configuration failed: $(jq -r '.msg // "unknown error"' "${response_file}")" >&2
  exit 1
fi

echo "SMTP configuration saved for ${OMC_URL}."
echo "Apply batch: $(jq -r '.data.batch.id // "n/a"' "${response_file}")"

if [[ "${SEND_TEST_EMAIL}" == "true" ]]; then
  jq -n --arg recipient "${TEST_RECIPIENT}" '{recipient:$recipient}' >"${test_payload_file}"
  sed \
    -e "s#${api_base}/admin/sysConfig/batch#${api_base}/admin/notification/email/test#" \
    -e "s#@${payload_file}#@${test_payload_file}#" \
    -e "s#${response_file}#${test_response_file}#" \
    "${curl_config}" >"${curl_config}.test"
  mv "${curl_config}.test" "${curl_config}"
  curl --config "${curl_config}"
  if [[ "$(jq -r '.ret // 0' "${test_response_file}")" != "0" ]]; then
    echo "Test email failed: $(jq -r '.msg // "unknown error"' "${test_response_file}")" >&2
    exit 1
  fi
  echo "One test email sent only to ${TEST_RECIPIENT}."
fi
