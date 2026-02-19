#!/usr/bin/env bash
set -euo pipefail

ENDPOINT="${1:-http://127.0.0.1:8080/debug/vars}"

required_keys=(
  "api_requests_total"
  "api_requests_errors_total"
  "api_request_latency_ms_total"
  "api_request_latency_samples_total"
  "api_requests_by_route"
  "api_request_errors_by_route"
  "ws_connections_active"
  "ws_connections_total"
  "ws_auth_failures_total"
  "ws_session_init_failures_total"
)

echo "Verifying expvar metrics from: ${ENDPOINT}"
payload="$(curl --fail --silent --show-error "${ENDPOINT}")"

for key in "${required_keys[@]}"; do
  if ! jq -e --arg key "${key}" 'has($key)' >/dev/null <<<"${payload}"; then
    echo "ERROR: missing required key: ${key}"
    exit 1
  fi
done

requests_total="$(jq -r '.api_requests_total' <<<"${payload}")"
requests_errors_total="$(jq -r '.api_requests_errors_total' <<<"${payload}")"
latency_total_ms="$(jq -r '.api_request_latency_ms_total' <<<"${payload}")"
latency_samples="$(jq -r '.api_request_latency_samples_total' <<<"${payload}")"
ws_active="$(jq -r '.ws_connections_active' <<<"${payload}")"
ws_total="$(jq -r '.ws_connections_total' <<<"${payload}")"
ws_auth_failures="$(jq -r '.ws_auth_failures_total' <<<"${payload}")"
ws_session_init_failures="$(jq -r '.ws_session_init_failures_total' <<<"${payload}")"

avg_latency="n/a"
if [[ "${latency_samples}" != "0" ]]; then
  avg_latency="$(jq -n --arg total "${latency_total_ms}" --arg samples "${latency_samples}" '$total|tonumber / ($samples|tonumber)')"
fi

echo "OK: required metrics are present."
echo "api_requests_total=${requests_total}"
echo "api_requests_errors_total=${requests_errors_total}"
echo "api_avg_latency_ms=${avg_latency}"
echo "ws_connections_active=${ws_active}"
echo "ws_connections_total=${ws_total}"
echo "ws_auth_failures_total=${ws_auth_failures}"
echo "ws_session_init_failures_total=${ws_session_init_failures}"
