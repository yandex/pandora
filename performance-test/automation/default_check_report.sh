#!/usr/bin/env bash

_CHECK_JSON_FILE="$1"

rc=0

check_json_val \
    'report status is READY' \
    '.status' \
    '== "READY"'

rc=$((rc | $?))

check_json_val \
    'has successfully sent requests' \
    '.overall.net_codes."0" // "-1" | tonumber' \
    '> 0'

rc=$((rc | $?))

check_json_val \
    'has non-zero response time requests' \
    '.overall.quantiles.q100 // "-1" | tonumber' \
    '> 0'

rc=$((rc | $?))

check_json_val \
    '50th response time percentile is less than 10s' \
    '.overall.quantiles.q50 // "-1" | tonumber' \
    '< 10000'

rc=$((rc | $?))

exit $rc
