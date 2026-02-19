#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-http://127.0.0.1:8080}"
SAMPLE_SECONDS="${2:-60}"

if ! [[ "${SAMPLE_SECONDS}" =~ ^[0-9]+$ ]]; then
  echo "ERROR: sample_seconds must be an integer >= 0"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HEALTH_URL="${BASE_URL%/}/health"
VARS_URL="${BASE_URL%/}/debug/vars"

echo "Lean baseline verification target: ${BASE_URL}"
echo "1) Checking health endpoint: ${HEALTH_URL}"
health_body="$(curl --fail --silent --show-error "${HEALTH_URL}")"
if [[ "${health_body}" != "ok" ]]; then
  echo "ERROR: /health returned unexpected body: ${health_body}"
  exit 1
fi
echo "OK: /health returned 200 with body 'ok'."

echo "2) Checking required /debug/vars keys"
"${SCRIPT_DIR}/verify-debug-vars.sh" "${VARS_URL}" >/dev/null
echo "OK: required /debug/vars keys are present."

snapshot() {
  curl --fail --silent --show-error "${VARS_URL}" | jq -c '{
    api_requests_total,
    api_requests_errors_total,
    api_request_latency_ms_total,
    api_request_latency_samples_total
  }'
}

if [[ "${SAMPLE_SECONDS}" == "0" ]]; then
  echo "3) Skipping sampled rate calculation (sample_seconds=0)."
  echo "LEAN BASELINE CHECK: PASS (endpoint and metrics reachability)"
  exit 0
fi

echo "3) Sampling counters for ${SAMPLE_SECONDS}s to estimate error/latency trends"
start="$(snapshot)"
sleep "${SAMPLE_SECONDS}"
end="$(snapshot)"

api_req_delta="$(jq -n --argjson s "${start}" --argjson e "${end}" '($e.api_requests_total - $s.api_requests_total)')"
api_err_delta="$(jq -n --argjson s "${start}" --argjson e "${end}" '($e.api_requests_errors_total - $s.api_requests_errors_total)')"
lat_ms_delta="$(jq -n --argjson s "${start}" --argjson e "${end}" '($e.api_request_latency_ms_total - $s.api_request_latency_ms_total)')"
lat_samples_delta="$(jq -n --argjson s "${start}" --argjson e "${end}" '($e.api_request_latency_samples_total - $s.api_request_latency_samples_total)')"

error_rate="$(jq -n --arg req "${api_req_delta}" --arg err "${api_err_delta}" 'if ($req|tonumber) <= 0 then 0 else (($err|tonumber)/($req|tonumber)) end')"
avg_latency_ms="$(jq -n --arg total "${lat_ms_delta}" --arg samples "${lat_samples_delta}" 'if ($samples|tonumber) <= 0 then 0 else (($total|tonumber)/($samples|tonumber)) end')"

echo "Sample summary (${SAMPLE_SECONDS}s):"
echo "api_requests_delta=${api_req_delta}"
echo "api_errors_delta=${api_err_delta}"
echo "api_error_rate=${error_rate}"
echo "api_avg_latency_ms=${avg_latency_ms}"

if [[ "${api_req_delta}" -ge 20 ]] && jq -e --arg er "${error_rate}" '$er|tonumber >= 0.05' >/dev/null; then
  echo "WARNING: Lean alert threshold exceeded (error_rate>=5% with >=20 requests in sample window)."
fi

echo "LEAN BASELINE CHECK: PASS (signal reachability confirmed)"
