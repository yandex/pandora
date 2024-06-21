#!/usr/bin/env bash

_metafile=''
_config_ids=()
_extra_description=''
_extra_labels=''
_extra_agent_filter=''
_extra_data_fnames=()
_extra_data_fspecs=()
while [[ $# -gt 0 ]]; do
    case "$1" in
    -m | --meta)
        [[ -n $2 ]]
        _metafile=$2
        shift
        shift
        ;;
    -c | --config-id)
        [[ -n $2 ]]
        _config_ids+=("$2")
        shift
        shift
        ;;
    -d | --extra-test-data)
        [[ -n $2 && -n $3 && -n $4 ]]
        _extra_data_fnames+=("$2")
        _extra_data_fspecs+=("name=$2,s3file=$3,s3bucket=$4")
        shift
        shift
        shift
        shift
        ;;
    --extra-labels)
        [[ -n $2 ]]
        [[ -z $_extra_labels ]]
        _extra_labels=$2
        shift
        shift
        ;;
    --extra-agent-filter)
        [[ -n $2 ]]
        [[ -z $_extra_agent_filter ]]
        _extra_agent_filter=$2
        shift
        shift
        ;;
    --extra-description)
        [[ -n $2 ]]
        [[ -z $_extra_description ]]
        _extra_description=$2
        shift
        shift
        ;;
    --help | -h | *)
        echo "Usage: $(basename "$0") [-m META_JSON_FILE] [-c CONFIG_ID]... [-d FILE_LOCAL_NAME FILE_BUCKET_NAME BUCKET_NAME]... [... OPTIONS]"
        echo ""
        echo "Compose arguments for 'yc loadtesting test create'."
        echo " -m|--meta META_JSON_FILE - path to a json file with test description"
        echo " -c|--config-id CONFIG_ID - ID of a test configuration file (may be defined multiple times)"
        echo " -d|--extra-test-data FILE_NAME FILE_NAME_IN_BUCKET BUCKET - extra test data (may be defined multiple times)"
        echo " --extra-labels KEY1=VAL1[,KEYN=VALN] - extra labels"
        echo " --extra-agent-filter AGENT_FILTER - extra agent filter"
        echo " --extra-description DESCRIPTION - extra description"
        exit 0
        ;;
    esac
done

_multi_factor="1"
_name=$(readlink -f "$_metafile" | xargs dirname | xargs basename)
_description=''
_labels=''
_agent_filter=''
_data_fnames=()
_data_fspecs=()

# ---------------------------------------------------------------------------- #
#                              read from metafile                              #
# ---------------------------------------------------------------------------- #

if [[ -f "$_metafile" ]]; then
    function read_meta {
        jq -re "$@" < "$_metafile"
        return 0
    }

    # shellcheck disable=SC2016
    _multi_factor=$(read_meta --arg d "$_multi_factor" '
        .multi // $d 
        | tostring 
    ')
    # shellcheck disable=SC2016
    _name=$(read_meta --arg d "$_name" '
        .name // $d 
        | tostring 
    ')
    _description=$(read_meta '
        .description // "" 
        | tostring 
    ')
    _labels=$(read_meta '
        .labels // {} 
        | to_entries 
        | map("\(.key)=\(.value)") 
        | join(",")
    ')
    _agent_filter=$(read_meta '
        .agent_labels // {} 
        | to_entries
        | map("labels.\(.key)=\"\(.value)\"") 
        | join(" and ")
    ')
    IFS=$'\n' read -d '' -ra _data_fnames < <(read_meta '
        .external_data // []
        | [.[] | select(.name? and .s3file? and .s3bucket?)]
        | map("\(.name)")
        | join("\n")
    ') || true
    IFS=$'\n' read -d '' -ra _data_fspecs < <(read_meta '
        .external_data // []
        | [.[] | select(.name? and .s3file? and .s3bucket?)]
        | map("name=\(.name),s3file=\(.s3file),s3bucket=\(.s3bucket)")
        | join("\n")
    ') || true
fi

# ---------------------------------------------------------------------------- #
#                                  add extras                                  #
# ---------------------------------------------------------------------------- #

if [[ -n ${_extra_description} ]]; then
    if [[ -n ${_description} ]]; then
        _description="$_description\n\n$_extra_description"
    else
        _description="$_extra_description"
    fi
fi

if [[ -n ${_extra_labels} ]]; then
    if [[ -n ${_labels} ]]; then
        _labels="$_labels,$_extra_labels"
    else
        _labels="$_extra_labels"
    fi
fi

if [[ -n ${_extra_agent_filter} ]]; then
    if [[ -n ${_agent_filter} ]]; then
        _agent_filter="$_agent_filter and $_extra_agent_filter"
    else
        _agent_filter="$_extra_agent_filter"
    fi
fi

_data_fnames+=("${_extra_data_fnames[@]}")
_data_fspecs+=("${_extra_data_fspecs[@]}")

# ---------------------------------------------------------------------------- #
#                                 compose args                                 #
# ---------------------------------------------------------------------------- #

ARGS=()
ARGS+=(--name "$_name")
ARGS+=(--description "$_description")
ARGS+=(--labels "$_labels")
for _ in $(seq 1 "$_multi_factor"); do
    for _config_id in "${_config_ids[@]}"; do
        _cfg="id=$_config_id,agent-by-filter=$_agent_filter"
        for _fname in "${_data_fnames[@]}"; do
            _cfg="$_cfg,test-data=$_fname"
        done

        ARGS+=(--configuration "$_cfg")
    done
done
for _fspec in "${_data_fspecs[@]}"; do
    ARGS+=(--test-data "$_fspec")
done

(IFS=$'\t'; echo "${ARGS[*]}")
