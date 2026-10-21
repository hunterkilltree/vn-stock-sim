#!/usr/bin/env bash
# Saves raw VCI fundamentals responses as test fixtures for the planned
# symbol/vcifinancials.go adapter (phase-fundamentals-data.md step 0).
# Run on a machine that can reach *.vietcap.com.vn, from backend/:
#   ./scripts/capture-vci-fundamentals.sh [SYMBOL ...]
set -euo pipefail

SYMBOLS=("${@:-VCB FPT HPG}")
read -r -a SYMBOLS <<<"${SYMBOLS[*]}"
OUT=internal/symbol/testdata/vci
IQ=https://iq.vietcap.com.vn/api/iq-insight-service
UA="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"
JAR=$(mktemp)
trap 'rm -f "$JAR"' EXIT
mkdir -p "$OUT"

get() { # get <file> <url>
  local code
  code=$(curl -sS -m 30 -b "$JAR" -c "$JAR" -o "$OUT/$1" -w '%{http_code}' \
    -H "Accept: application/json, text/plain, */*" \
    -H "Referer: https://trading.vietcap.com.vn/" -H "Origin: https://trading.vietcap.com.vn" \
    -H "User-Agent: $UA" "$2") || code=ERR
  echo "$code  $1"
  sleep 0.5
}

# vnstock does this handshake first; keep its cookies in case IQ needs them.
curl -sS -m 30 -c "$JAR" -o /dev/null -H "User-Agent: $UA" https://trading.vietcap.com.vn/priceboard || true

today=$(date +%Y%m%d)
for s in "${SYMBOLS[@]}"; do
  get "$s-income.json"     "$IQ/v1/company/$s/financial-statement?section=INCOME_STATEMENT"
  get "$s-balance.json"    "$IQ/v1/company/$s/financial-statement?section=BALANCE_SHEET"
  get "$s-metrics.json"    "$IQ/v1/company/$s/financial-statement/metrics"
  get "$s-statistics.json" "$IQ/v1/company/$s/statistics-financial"
  get "$s-details.json"    "$IQ/v1/company/details?ticker=$s"
  get "$s-events.json"     "$IQ/v1/events?ticker=$s&fromDate=20150101&toDate=$today&eventCode=DIV,ISS&page=0&size=200"
done

# Daily bars for FPT since 2015: compare closes around its ISS ex-dates in
# FPT-events.json to tell whether gap-chart prices are adjusted.
code=$(curl -sS -m 30 -o "$OUT/FPT-bars-1D.json" -w '%{http_code}' -X POST \
  -H "Content-Type: application/json" -H "Referer: https://trading.vietcap.com.vn/" \
  -H "Origin: https://trading.vietcap.com.vn/" -H "User-Agent: $UA" \
  -d "{\"timeFrame\":\"ONE_DAY\",\"symbols\":[\"FPT\"],\"to\":$(date +%s),\"countBack\":3000}" \
  https://trading.vietcap.com.vn/api/chart/OHLCChart/gap-chart) || code=ERR
echo "$code  FPT-bars-1D.json"
