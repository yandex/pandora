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
while [[ $# -gt 0 ]]; do
    case "$1" in
    -h | --help)
        echo "Usage: $(basename "$0") [ARG]..."
        echo ""
        echo "Create an agent and wait until it is READY_FOR_TEST"
        echo ""
        echo "Provided arguments are passed to 'yc loadtesting agent create [ARG]...' as is."
        echo "If missing, some argument values are defaulted YC_LT_* environment variables."
        exit 0
        ;;
    --service-account-id)
        VAR_AGENT_SA_ID=$2
        shift
        shift
        ;;
    --name)
        _AGENT_NAME=$2
        shift
        shift
        ;;
    --description)
        VAR_AGENT_DESCRIPTION=$2
        shift
        shift
        ;;
    --labels)
        VAR_AGENT_LABELS=$2
        shift
        shift
        ;;
    --zone)
        VAR_AGENT_ZONE=$2
        shift
        shift
        ;;
    --cores)
        VAR_AGENT_CORES=$2
        shift
        shift
        ;;
    --memory)
        VAR_AGENT_MEMORY=$2
        shift
        shift
        ;;
    --network-interface)
        VAR_AGENT_SUBNET_ID=
        VAR_AGENT_SECURITY_GROUP_IDS=
        _ARGS+=(--network-interface "$2")
        shift
        shift
        ;;
    --)
        shift
        ;;
    *)
        _ARGS+=("$1")
        shift
        ;;
    esac
done

assert_installed yc jq
assert_not_empty YC_LT_AGENT_SA_ID
assert_not_empty YC_LT_AGENT_SUBNET_ID
assert_not_empty YC_LT_AGENT_SECURITY_GROUP_IDS

if [[ -z "$_AGENT_NAME" ]]; then
    _AGENT_NAME="$VAR_AGENT_NAME_PREFIX$(rand_str)"
fi

# ---------------------------------------------------------------------------- #
#                               Assert variables                               #
# ---------------------------------------------------------------------------- #

if [[ -z "${VAR_FOLDER_ID:-$(yc_ config get folder-id)}" ]]; then
    _log "Folder ID must be specified either via YC_LT_FOLDER_ID or via CLI profile."
    exit 1
fi

# ---------------------------------------------------------------------------- #
#                         Compose command line options                         #
# ---------------------------------------------------------------------------- #

if [[ -n $_AGENT_NAME ]]; then
    _ARGS+=(--name "$_AGENT_NAME")
fi
if [[ -n $VAR_AGENT_SA_ID ]]; then
    _ARGS+=(--service-account-id "$VAR_AGENT_SA_ID")
fi
if [[ -n $VAR_AGENT_DESCRIPTION ]]; then
    _ARGS+=(--description "$VAR_AGENT_DESCRIPTION")
fi
if [[ -n $VAR_AGENT_LABELS ]]; then
    _ARGS+=(--labels "$VAR_AGENT_LABELS")
fi
if [[ -n $VAR_AGENT_ZONE ]]; then
    _ARGS+=(--zone "$VAR_AGENT_ZONE")
fi
if [[ -n $VAR_AGENT_CORES ]]; then
    _ARGS+=(--cores "$VAR_AGENT_CORES")
fi
if [[ -n $VAR_AGENT_MEMORY ]]; then
    _ARGS+=(--memory "$VAR_AGENT_MEMORY")
fi
if [[ -n ${VAR_AGENT_SUBNET_ID} || -n ${VAR_AGENT_SECURITY_GROUP_IDS} ]]; then
    _ARGS+=(--network-interface)
    _ARGS+=("subnet-id=$VAR_AGENT_SUBNET_ID,security-group-ids=$VAR_AGENT_SECURITY_GROUP_IDS")
fi

# ---------------------------------------------------------------------------- #
#                                Create an agent                               #
# ---------------------------------------------------------------------------- #

_log_stage "[$_AGENT_NAME]"
_log_push_stage "[CREATE]"

_log "Creating..."

if ! _agent=$(yc_lt agent create "${_ARGS[@]}"); then
    _log "Failed to create an agent. $_agent"
    exit 1
fi

_agent_id=$(echo "$_agent" | jq -r '.id')
_log "Agent created. id=$_agent_id"

# ---------------------------------------------------------------------------- #
#                      Wait until agent is READY_FOR_TEST                      #
# ---------------------------------------------------------------------------- #

_log_stage "[WAIT]"
_log "Waiting for agent to be ready..."

_TICK="5"
_TIMEOUT="600"

_ts_start=$(date +%s)
_ts_timeout=$((_ts_start + _TIMEOUT))
while [ "$(date +%s)" -lt $_ts_timeout ]; do
    _elapsed=$(($(date +%s) - _ts_start))

    if ! _status=$(yc_lt agent get "$_agent_id" | jq -r '.status'); then
        _log "Failed to get agent status"
        sleep "$_TICK"
        continue
    fi

    if [[ "$_status" == "READY_FOR_TEST" ]]; then
        _log_stage "[READY]"
        _logv 1 "Wow! Just ${_elapsed}s!"
        _log "READY_FOR_TEST"

        echo "$_agent_id"
        exit 0
    fi

    if ((_elapsed % (_TICK * 6) == 0)); then
        _logv 1 "${_elapsed}s passed. Status is $_status. Waiting..."
    fi

    _logv 2 "$_status. Next check in ${_TICK}s"
    sleep "$_TICK"
done

echo "$_agent_id"

_log_stage "[WAIT_FAILED]"
_log "STATUS=$_status. Timeout of ${_TIMEOUT}s exceeded"
_log "Agent is not ready and likely cant be used in tests!"
exit 1
