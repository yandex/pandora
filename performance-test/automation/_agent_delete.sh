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

_NAME_SUBSTRING="$VAR_AGENT_NAME_PREFIX"
_LABELS="$VAR_AGENT_LABELS"
_IDS=()
while [[ $# -gt 0 ]]; do
    case "$1" in
    -h | --help)
        echo "Usage: $(basename "$0") [--name-substring] [--labels LABELS] [ID1 [...IDN]]"
        echo ""
        echo "Delete agents matching given parameters."
        echo " --name-substring NAME_SUBSTRING - delete agents with name containing this substring [default env YC_LT_AGENT_NAME_PREFIX]"
        echo " --labels KEY1=VAL1[,KEYN=VALN] - delete agents with these labels [default env YC_LT_AGENT_LABELS]"
        echo " ID1 [...IDN] - agent's id must be one of the specified"
        exit 0
        ;;
    --name-substring)
        _NAME_SUBSTRING=$2
        shift
        shift
        ;;
    --labels)
        _LABELS=$2
        shift
        shift
        ;;
    *)
        _IDS+=("$1")
        shift
        ;;
    esac
done

assert_installed yc jq curl

if [[ -z $_NAME_SUBSTRING && -z $_LABELS && -z ${_IDS[*]} ]]; then
    _log "Cannot pick an agent to delete. At least one of arguments must be specified."
    exit 1
fi

# ---------------------------------------------------------------------------- #
#                                Compose filter                                #
# ---------------------------------------------------------------------------- #

_filters=()
if [[ -n $_NAME_SUBSTRING ]]; then
    _filters+=("name contains \"$_NAME_SUBSTRING\"")
fi
if [[ -n $_LABELS ]]; then
    IFS=',' read -ra _labels_arr <<<"$_LABELS"
    for _kv in "${_labels_arr[@]}"; do
        IFS='=' read -r _key _value <<<"$_kv"
        _filters+=("labels.$_key = \"$_value\"")
    done
fi
if [[ -n ${_IDS[*]} ]]; then
    _joined=$(IFS=','; echo "${_IDS[*]}")
    _filter+=("id in ($_joined)")
fi

_filter_str=''
for _f in "${_filters[@]}"; do
    if [[ -n $_filter_str ]]; then
        _filter_str="$_filter_str and $_f"
    else
        _filter_str="$_f"
    fi
done

if [[ -z $_filter_str ]]; then
    _log "Error! Filter is empty"
    exit 1
fi

_log "Filter: $_filter_str"

# ---------------------------------------------------------------------------- #
#                   Determine which agents should be deleted                   #
# ---------------------------------------------------------------------------- #

_log "Determining which agents to be deleted..."


_delete_ids=()
IFS=' ' read -ra _delete_ids < \
    <(yc_lt agent list --filter "$_filter_str" | jq -r '[.[].id] | join(" ")')

if [[ ${#_delete_ids} -eq 0 ]]; then
    _log "No agents were found for given filter"
    exit 0
fi

_log "Agents to be deleted: ${_delete_ids[*]}"

# ---------------------------------------------------------------------------- #
#                                 Delete agents                                #
# ---------------------------------------------------------------------------- #

_log "Deleting agents..."
yc_lt agent delete "${_delete_ids[@]}"
