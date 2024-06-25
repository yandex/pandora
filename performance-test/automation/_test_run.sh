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
    --help | -h)
        echo "Usage: $(basename "$0") TEST_DIR"
        echo ""
        echo "Run test with configurations defined in TEST_DIR/$VAR_TEST_CONFIG_MASK"
        echo "Additional test parameters may be defined in TEST_DIR/meta.json"
        exit 0
        ;;
    *)
        _TEST_DIR=$1
        break
        ;;
    esac
done

assert_installed yc jq curl
assert_not_empty _TEST_DIR

_TEMP_BUCKET_DIR="test-runs/$(rand_str)"
declare -r _TEMP_BUCKET_DIR

# ---------------------------------------------------------------------------- #
#                   sanity check, before anything is created                   #
# ---------------------------------------------------------------------------- #

_logv 1 "## Sanity check..."
run_script "$_SCRIPT_DIR/_compose_test_create_args.sh" \
    --meta "$_TEST_DIR/meta.json" \
    -c 12345 \
    -c 54321 \
    -d local1 inbucket1 bucket1 \
    -d local2 inbucket2 bucket2 \
    >/dev/null

# ---------------------------------------------------------------------------- #
#                       prepare test configuration files                       #
# ---------------------------------------------------------------------------- #

_logv 1 "## Prepare test configurations"
_config_ids=()

# ------------------------- list configuration files ------------------------- #

_config_files=()
while IFS= read -d '' -r _file; do _config_files+=("$_file"); done < \
    <(find "$_TEST_DIR" -type f -name "$VAR_TEST_CONFIG_MASK" -maxdepth 1 -print0)

if [[ ${#_config_files[@]} -eq 0 ]]; then
    _log "ERROR!!! No config files found in $_TEST_DIR. Config file mask: $VAR_TEST_CONFIG_MASK"
    exit 1
fi

_logv 1 "Found test configuration files: ${_config_files[*]}"

# ------------------------ upload configuration files ------------------------ #

_logv 1 "Uploading configurations..."
for _file in "${_config_files[@]}"; do
    _args=()

    # substitute YC_LT_TARGET in config file if the variable is set
    if [[ -n $YC_LT_TARGET ]]; then
        _config_content=$(cat "$_file")
        # shellcheck disable=SC2016
        _config_content=${_config_content/'${YC_LT_TARGET}'/"$YC_LT_TARGET"}
        _args+=(--yaml-string "$_config_content")
    else
        _args+=(--from-yaml-file "$_file")
    fi

    _config_id=$(yc_lt test-config create "${_args[@]}" | jq -r '.id')
    _logv 1 "- created test configuration $_config_id from $_file"

    _config_ids+=("$_config_id")
done

# ---------------------------------------------------------------------------- #
#                         prepare local data files                             #
# ---------------------------------------------------------------------------- #

_logv 1 "## Prepare local data files"
_local_data_fnames=()

function cleanup_temp_data_files {
    _log "Cleaning up data files..."
    for _fname in "${_local_data_fnames[@]}"; do
        _temp_s3_file="$_TEMP_BUCKET_DIR/$_fname"
        if ! yc_s3_delete "$_temp_s3_file" "$VAR_DATA_BUCKET" >/dev/null; then
            _log "- failed to delete $_temp_s3_file"
        fi
    done
}
trap cleanup_temp_data_files EXIT

# --------------------------- list local data files -------------------------- #

function is_data_file {
    _non_data_files=("${_config_files[@]}")
    _non_data_files+=("$_TEST_DIR/meta.json")
    _non_data_files+=("$_TEST_DIR/check_summary.sh")
    _non_data_files+=("$_TEST_DIR/check_report.sh")
    for _ndf in "${_non_data_files[@]}"; do
        if [[ $1 == "$_ndf" ]]; then
            return 1
        fi
    done
    return 0
}

_local_data_files=()
while IFS= read -d '' -r _file; do
    if is_data_file "$_file"; then
        _local_data_files+=("$_file")
    fi
done < <(find "$_TEST_DIR" -type f -print0)

_logv 1 "Found local data files: ${_local_data_files[*]}"

# --------------------- upload local data files to bucket -------------------- #

if [[ ${#_local_data_files[@]} -gt 0 && -n $VAR_DATA_BUCKET ]]; then
    _logv 1 "Uploading local data files... (should be deleted after test)"
    _logv 1 "upload params: bucket=$VAR_DATA_BUCKET; common-prefix=$_TEMP_BUCKET_DIR/"

    for _file in "${_local_data_files[@]}"; do
        _fname=${_file#"$_TEST_DIR/"}
        _temp_s3_file="$_TEMP_BUCKET_DIR/$_fname"
        if ! yc_s3_upload "$_file" "$_temp_s3_file" "$VAR_DATA_BUCKET" >/dev/null; then
            _log "- failed to upload $_temp_s3_file"
            continue
        fi

        _logv 1 "- uploaded local data file $_fname"
        _local_data_fnames+=("$_fname")
    done

elif [[ ${#_local_data_files[@]} -gt 0 ]]; then
    _logv 1 "Upload failed: YC_LT_DATA_BUCKET is not specified."
fi

# ---------------------------------------------------------------------------- #
#                           Determine test parameters                          #
# ---------------------------------------------------------------------------- #

_logv 1 "## Compose command arguments"

_composer_args=()
_composer_args+=(--meta "$_TEST_DIR/meta.json")
_composer_args+=(--extra-agent-filter "${VAR_TEST_AGENT_FILTER:-}")
_composer_args+=(--extra-labels "${VAR_TEST_EXTRA_LABELS:-}")
_composer_args+=(--extra-description "${VAR_TEST_EXTRA_DESCRIPTION:-}")
for _id in "${_config_ids[@]}"; do
    _composer_args+=(-c "$_id")
done
for _fname in "${_local_data_fnames[@]}"; do
    _composer_args+=(-d "$_fname" "$_TEMP_BUCKET_DIR/$_fname" "$VAR_DATA_BUCKET")
done

RUN_ARGS=()
IFS=$'\t' read -d '' -ra RUN_ARGS < \
    <(run_script "$_SCRIPT_DIR/_compose_test_create_args.sh" "${_composer_args[@]}") \
    || true

_logv 1 "Test run arguments: ${RUN_ARGS[*]}"

# ---------------------------------------------------------------------------- #
#                                 Run the test                                 #
# ---------------------------------------------------------------------------- #

_logv 1 "## Starting test..."
_test_id=$(yc_lt test create "${RUN_ARGS[@]}" | jq -r '.id')

_logv 1 "Started. Test url: $(yc_test_url "$_test_id")/test-report"

_logv 1 "## Waiting for test to finish..."
yc_lt test wait --idle-timeout 60s "$_test_id"
