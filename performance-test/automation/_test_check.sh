#!/usr/bin/env bash

set -eo pipefail

# shellcheck disable=SC2155
export _SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

# shellcheck source=_functions.sh
source "$_SCRIPT_DIR/_functions.sh"

# shellcheck source=_variables.sh
source "$_SCRIPT_DIR/_variables.sh"

# ---------------------------------------------------------------------------- #
#                            Arguments and constants                           #
# ---------------------------------------------------------------------------- #

while [[ $# -gt 0 ]]; do
    case "$1" in
    --dir)
        _TEST_DIR=$2
        shift
        shift
        ;;
    --id)
        _TEST_ID=$2
        shift
        shift
        ;;
    --output | -o)
        _OUTPUT_DIR=$2
        shift
        shift
        ;;
    --help | -h | *)
        echo "Usage: $(basename "$0") --id TEST_ID [--dir SCRIPT_DIR] [-o OUTPUT_DIR]"
        echo ""
        echo "Obtain test results and check them with two check scripts in passed SCRIPT_DIR directory:"
        # shellcheck disable=SC2016
        echo '- SCRIPT_DIR/check_summary.sh $(yc --format json loadtesting test get TEST_ID)'
        # shellcheck disable=SC2016
        echo '- SCRIPT_DIR/check_report.sh $(yc --format json loadtesting test get-report-table TEST_ID)'
        echo ""
        echo "If corresponding checks are not found in SCRIPT_DIR, the default checks will be run instead."
        exit 0
        ;;
    esac
done

assert_installed yc jq curl
assert_not_empty _TEST_DIR
assert_not_empty _TEST_ID

_OUTPUT_DIR=${_OUTPUT_DIR:-"./check-$_TEST_DIR"}
mkdir -p "$_OUTPUT_DIR"

set +e
export -f run_script
export -f check_json_val

rc=0

# ------------------------------- Check summary ------------------------------ #
# -------------- (yc --format json loadtesting test get TEST_ID) ------------- #

if ! yc_lt test get "$_TEST_ID" >"$_OUTPUT_DIR/summary.json"; then
    echo "ERROR!!! failed to download test summary"
    exit 1
fi

export _DEFAULT_CHECK="$_SCRIPT_DIR/default_check_summary.sh"
if [[ -f "$_TEST_DIR/check_summary.sh" ]]; then
    echo "Running: $_TEST_DIR/check_summary.sh $_OUTPUT_DIR/summary.json"
    if ! /usr/bin/env bash "$_TEST_DIR/check_summary.sh" "$_OUTPUT_DIR/summary.json"; then
        rc=1
    fi

elif [[ -f "$_DEFAULT_CHECK" ]]; then
    echo "Running: $_DEFAULT_CHECK $_OUTPUT_DIR/summary.json"
    if ! /usr/bin/env bash "$_DEFAULT_CHECK" "$_OUTPUT_DIR/summary.json"; then
        rc=1
    fi

else
    echo "ERROR: check_summary.sh script not found"
    rc=1
fi

# ------------------------------- Check report ------------------------------- #
# ------- (yc --format json loadtesting test get-report-table TEST_ID) ------- #

if ! yc_lt test get-report-table "$_TEST_ID" >"$_OUTPUT_DIR/report.json"; then
    echo "ERROR!!! failed to download test report"
    exit 1
fi

export _DEFAULT_CHECK="$_SCRIPT_DIR/default_check_report.sh"
if [[ -f "$_TEST_DIR/check_report.sh" ]]; then
    echo "Running: $_TEST_DIR/check_report.sh $_OUTPUT_DIR/report.json"
    if ! /usr/bin/env bash "$_TEST_DIR/check_report.sh" "$_OUTPUT_DIR/report.json"; then
        rc=1
    fi

elif [[ -f "$_DEFAULT_CHECK" ]]; then
    echo "Running: $_DEFAULT_CHECK $_OUTPUT_DIR/report.json"
    if ! /usr/bin/env bash "$_DEFAULT_CHECK" "$_OUTPUT_DIR/report.json"; then
        rc=1
    fi

else
    echo "ERROR: check_summary.sh script not found"
    rc=1
fi

# ----------------------------------- exit ----------------------------------- #

exit $rc
