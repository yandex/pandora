#!/usr/bin/env bash

_CHECK_JSON_FILE="$1"

rc=0

check_json_val \
    'test status doesnt indicate an error' \
    '.summary.status' \
    '| IN("DONE", "AUTOSTOPPED")'

rc=$((rc | $?))

exit $rc
