#!/usr/bin/env bash

if [[ -n ${__YC_LT_VARS_DEFINED} ]]; then
    return 0
fi

export __YC_LT_VARS_DEFINED=1

function _def_var {
    local -r _name=$1

    local -r _env="YC_LT_${_name}"

    local -r _var_default="__DEFAULT_${_name}"
    local -r _var="VAR_${_name}"

    local -r _val_default=$2
    local -r _val=${!_env-"${_val_default}"}

    export "$_var_default=$_val_default"
    export "$_var=$_val"

    local -r _gap=$(printf -- '.%.0s' {1..80})
    local -r _info_actual="$_var: \"${!_var:0:60}\""
    local -r _info_env="$_env: \"${!_env:0:60}\""
    _logv 2 "${_info_actual} ${_gap:${#_info_actual}} env ${_info_env}"
}

# ---------------------------------------------------------------------------- #
#                                    General                                   #
# ---------------------------------------------------------------------------- #

# env: YC_LT_VERBOSE
# format: 0, 1, 2
# Scripts verbosity level
_def_var VERBOSE "0"

# env: YC_LT_OUTPUT_DIR
# format: path to local directory
_def_var OUTPUT_DIR "$PWD/.loadtesting"

# env: YC_LT_CLI_PROFILE
# format: string
# YC CLI profile which will be used to create agents and tests.
# An ID of a folder is taken from profile, or, if specified, from YC_LT_FOLDER_ID.
_def_var CLI_PROFILE ""

# env: YC_LT_FOLDER_ID
# format: string, cloud-id
# ID of a cloud folder where tests are performed.
_def_var FOLDER_ID ""

# env: YC_LT_DATA_BUCKET
# format: string, bucket-name
# Name of an object storage bucket used as a storage for test data. If needed,
# local files are automatically uploaded to the bucket.
_def_var DATA_BUCKET ""

# env: YC_LT_CLI_INTERACTIVE
# format: 0 or 1
# Defines whether interactive CLI input is allowed.
_def_var CLI_INTERACTIVE "1"

# env: YC_LT_TOKEN
# format: string, token
# A token. Normally, no need to specify explicitly
_def_var TOKEN ""

# ---------------------------------------------------------------------------- #
#                                     Agent                                    #
# ---------------------------------------------------------------------------- #

# env: YC_LT_AGENTS_CNT
# format: positive number
# Number of agents which should be created when 'agent.sh create' is called.
_def_var AGENTS_CNT "1"

# env: YC_LT_AGENT_SA_ID
# format: string, cloud-id
# ID of a service account with which agent VM will be created.
_def_var AGENT_SA_ID ""

# ---------------------------- Agent: VM settings ---------------------------- #

# env: YC_LT_AGENT_ZONE
# format: identifier of an availability zone
# Agent's VM will be created in the specified zone
_def_var AGENT_ZONE "ru-central1-b"

# env: YC_LT_AGENT_SUBNET_ID
# format: string, cloud-id
# ID of a subnet in which agent's VM will be created
_def_var AGENT_SUBNET_ID ""

# env: YC_LT_AGENT_SECURITY_GROUP_IDS
# format: list of cloud-id in format [id[,id[,...]]]
# IDS of security groups assigned to created agent.
_def_var AGENT_SECURITY_GROUP_IDS ""

# env: YC_LT_AGENT_CORES
# format: positive number > 1
# A number of CPU cores with which agent VM will be created.
_def_var AGENT_CORES "2"

# env: YC_LT_AGENT_MEMORY
# format: positive number + scale specifier
# Amount of RAM with which agent VM will be created.
_def_var AGENT_MEMORY "2G"

# ----------------- Agent: service settings and customization ---------------- #

# env: YC_LT_AGENT_LABELS
# format: list of key-value pairs in format [key=value[,key=value[,...]]]
# Labels of an agent created by 'agent.sh create'
_def_var AGENT_LABELS "ci=true,author=$USER"

# env: YC_LT_AGENT_NAME_PREFIX
# format: string
# Name (or prefix) of an agent created by 'agent.sh create'
_def_var AGENT_NAME_PREFIX "ci-lt-agent"

# env: YC_LT_AGENT_DESCRIPTION
# format: string
# Description of an agent created by 'agent.sh create'
_def_var AGENT_DESCRIPTION "Created via script by $USER"

# ---------------------------------------------------------------------------- #
#                                     Test                                     #
# ---------------------------------------------------------------------------- #

# env: YC_LT_SKIP_TEST_CHECK
# format: 0 or 1
# Specifies whether checks should be performed after a test has finished.
_def_var SKIP_TEST_CHECK "0"

# env: YC_LT_TEST_AGENT_FILTER
# format: filter string
# Filter expression by which agents will be selected to execute a test.
# The expression will be ANDed to the ones specified in meta.json
# Example:
#  - agents containing 'onetime-agent-' in name: 'name contains "onetime-agent-"'
#  - agents with labels ci=true and author=foobar: 'labels.ci = "true" and labels.author = "foobar"'
_def_var TEST_AGENT_FILTER "labels.ci=true and labels.author=$USER"

# env: YC_LT_TEST_EXTRA_LABELS
# format: list of key-value pairs in format [key=value[,key=value[,...]]]
# Additional (to the ones specified in meta.json) labels with which tests will be created.
_def_var TEST_EXTRA_LABELS "ci=true"

# env: YC_LT_TEST_EXTRA_DESCRIPTION
# format: string
# Additional (to the one specified in meta.json) description which which tests will be creatd.
_def_var TEST_EXTRA_DESCRIPTION ""

# ---------------------------------------------------------------------------- #
#                            Constants customization                           #
# ---------------------------------------------------------------------------- #

# env: YC_LT_TEST_CONFIG_MASK
# format: GLOB mask
_def_var TEST_CONFIG_MASK "test-config*.yaml"

# env: YC_LT_OBJECT_STORAGE_URL
# format: url
_def_var OBJECT_STORAGE_URL "https://storage.yandexcloud.net"

# env: YC_LT_WEB_CONSOLE_URL
# format: url
_def_var WEB_CONSOLE_URL "https://console.yandex.cloud"
