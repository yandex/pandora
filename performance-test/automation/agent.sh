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

_CMD=''

while [[ $# -gt 0 ]]; do
    case "$1" in
    create)
        _CMD='create'
        shift
        break
        ;;
    delete)
        _CMD='delete'
        shift
        break
        ;;
    -h | --help | *)
        echo "Usage: $(basename "$0") subcommand [ARG]..."
        echo ""
        echo "Subcommands:"
        echo " $(basename "$0") create [--count N] [ARG]..."
        echo "   create specified number of agents"
        echo " $(basename "$0") delete [ARG]..."
        echo "   delete agents"
        exit 0
        ;;
    esac
done

if [[ -z "${VAR_FOLDER_ID:-$(yc_ config get folder-id)}" ]]; then
    _log "Folder ID must be specified either via YC_LT_FOLDER_ID or via CLI profile."
    exit 1
fi

if [[ "$_CMD" == 'create' ]]; then
    _CNT=$VAR_AGENTS_CNT
    while [[ $# -gt 0 ]]; do
        case "$1" in
        -h | --help)
            echo "Usage: $(basename "$0") create [--count N] [ARG]..."
            echo ""
            echo "Call agent creation subroutine N times and wait until all agents are READY_FOR_TEST"
            echo ""
            echo "Subroutine help:"
            run_script "$_SCRIPT_DIR/_agent_create.sh" --help
            exit 0
            ;;
        --count)
            _CNT="$2"
            shift
            shift
            break
            ;;
        --)
            shift
            break
            ;;
        *)
            break
            ;;
        esac
    done

    _log "Compute Agents create request. Number of agents: $_CNT"
    _pids=()
    for _i in $(seq 1 "$_CNT"); do
        _log_stage "[$_i]"
        run_script "$_SCRIPT_DIR/_agent_create.sh" "$@" &
        _pids+=("$!")
    done

    _rc=0
    for _pid in "${_pids[@]}"; do
        wait "$_pid"
        _rc=$((_rc | $?))
    done

    exit ${_rc}

elif [[ "$_CMD" == 'delete' ]]; then
    while [[ $# -gt 0 ]]; do
        case "$1" in
        -h | --help)
            echo "Usage: $(basename "$0") delete [ARG]..."
            echo ""
            echo "Call agent deletion subroutine"
            echo ""
            echo "Subroutine help:"
            run_script "$_SCRIPT_DIR/_agent_delete.sh" --help
            exit 0
            ;;
        --)
            shift
            break
            ;;
        *)
            break
            ;;
        esac
    done

    _log "Compute Agents delete request."
    run_script "$_SCRIPT_DIR/_agent_delete.sh" "$@"
    exit $?
fi
