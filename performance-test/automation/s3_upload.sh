#!/usr/bin/env bash

set -eo pipefail

# shellcheck disable=SC2155
export _SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

# shellcheck source=_functions.sh
source "$_SCRIPT_DIR/_functions.sh"

# shellcheck source=_variables.sh
source "$_SCRIPT_DIR/_variables.sh"

# ---------------------------------------------------------------------------- #
#                     Retrieve arguments from command line                     #
# ---------------------------------------------------------------------------- #

_ARGS=()


_file="${1}"
_temp_s3_file="${2}"
var_data_bucket="${3}"

if ! yc_s3_upload "$_file" "$_temp_s3_file" "$var_data_bucket" >/dev/null; then
    _log "- failed to upload $_temp_s3_file"
    exit 1
fi
