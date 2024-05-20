[Home](../index.md)

---

# Scenario generator / gRPC

- [Configuration](#configuration)
    - [Generator](#generator)
    - [Provider](#provider)
- [Description of the scenario format](#description-of-the-scenario-format)
    - [General principle](#general-principle)
    - [HCL example](#hcl-example)
    - [YAML example](#yaml-example)
    - [Locals](#locals)
- [Features](#features)
    - [Calls](#calls)
        - [Templater](#templater)
            - [Variable names in templates](#variable-names-in-templates)
            - [Functions in templates](#functions-in-templates)
        - [Preprocessors](#preprocessors)
            - [prepare](#prepare)
        - [Postprocessors](#postprocessors)
            - [assert/response](#assertresponse)
    - [Scenarios](#scenarios)
    - [Sources](#sources)
- [References](#references)

## Configuration

You need to use a generator and a provider of type `grpc/scenario`

```yaml
pools:
  - id: Pool name
    gun:
      type: grpc/scenario
      target: localhost:8888
    ammo:
      type: grpc/scenario
      file: payload.hcl
```

### Generator

The minimum generator configuration is as follows

```yaml
gun:
  type: grpc/scenario
  target: localhost:8888
```

For a scenario gRPC generator, all settings of a regular gRPC generator are supported [gRPC generator](grpc-generator.md)

### Provider

The provider accepts only one parameter - the path to the file with the scenario description

```yaml
ammo:
  type: http/scenario
  file: payload.hcl
```

Supports file extensions

- hcl
- yaml
- json

## Description of the scenario format

Supports formats

- hcl
- yaml
- json

### General principle

Several scenarios can be described in a single file. A script has a name by which one scenario differs from another.

A script is a sequence of rpc calls. That is, you will need to describe in the script which calls
should be executed in what order.

The Call is a gRPC call. It has standard gRPC call fields plus additional ones. See [Calls](#calls).

### HCL example

```terraform
locals {
  common_meta = {
    "metadata" = "server.proto"
  }
  next = "next"
}
locals {
  auth_meta = merge(local.common_meta, {
    authorization = "{{.request.auth_req.postprocessor.token}}"
  })
  next = "next"
}
variable_source "users" "file/csv" {
  file              = "users.csv"
  fields            = ["user_id", "login", "pass"]
  ignore_first_line = true
  delimiter         = ","
}
variable_source "filter_src" "file/json" {
  file = "filter.json"
}
variable_source "variables" "variables" {
  variables = {
    header = "yandex"
    b      = "s"
  }
}

call "auth_req" {
  call     = "target.TargetService.Auth"
  tag      = "auth"
  metadata = local.auth_meta
  preprocessor "prepare" {
    mapping = {
      user = "source.users[next]"
    }
  }
  payload = <<EOF
{"login": "{{.request.auth_req.preprocessor.user.login}}", "pass": "{{.request.auth_req.preprocessor.user.pass}}"}
EOF
  postprocessor "assert/response" {
    payload     = ["token"]
    status_code = 200
  }
}

scenario "scenario_name" {
  weight           = 1
  min_waiting_time = 1000
  requests         = [
    "auth_req",
  ]
}
```

You can also see an example in the tests https://github.com/yandex/pandora/blob/dev/tests/grpc_scenario/testdata/grpc_payload.hcl


### YAML example

```yaml
locals:
  my-meta: &global-meta
    metadata: "server.proto"
variable_sources:
  - type: "file/csv"
    name: "users"
    ignore_first_line: true
    delimiter: ","
    file: "file.csv"
    fields: ["user_id", "login", "pass"]
  - type: "file/json"
    name: "filter_src"

calls:
  - name: "auth_req"
    call: 'target.TargetService.Auth'
    tag: auth
    method: POST
    metadata:
      <<: *global-meta
    preprocessors:
      - type: prepare
        mapping:
            new_var: source.var_name[next].0
    payload: '{"login": "{{.request.auth_req.preprocessor.user.login}}", "pass": "{{.request.auth_req.preprocessor.user.pass}}"}'
    postprocessors:
      - type: assert/response
        payload: ["token"]
        status_code: 200

scenarios:
  - name: scenario_name
    weight: 1
    min_waiting_time: 1000
    requests: [
      auth_req
    ]
```

### Locals

Про блок locals смотрите в отдельной [статье Locals](scenario/locals.md)

## Features

### Calls

Fields

- call
- tag
- metadata
- preprocessors
- payload
- postprocessors

### Templater

The fields `metadata', `payload` are templated.

The standard go template is used.

#### Variable names in templates

Variable names have the full path of their definition.

For example

Variable `users` from source `user_file` - `{% raw %}{{{.source.user_file.users}}{% endraw %}`

Variable `item` from the `list_req` call preprocessor - `{% raw %}{{{.request.list_req.preprocessor.item}}{% endraw %}`

> Note
> To maintain similarity with http scripts, the response section from the grpc call is saved to the `postprocessor` section

Variable `token`  from the `list_req` call is `{% raw %}{{{.request.list_req.postprocessor.token}}{% endraw %}`

##### Functions in Templates

Since the standard Go templating engine is used, it is possible to use built-in functions available at https://pkg.go.dev/text/template#hdr-Functions.

Additionally, some functions include:

- randInt
- randString
- uuid

For more details about randomization functions, see [more](scenario/functions.md).

#### Preprocessors

Preprocessor - actions are performed before templating

##### prepare

It is used for creating new variable mapping

The preprocessor has the ability to work with arrays using modifiers

- next
- last
- rand

##### yaml

```yaml
calls:
  - name: req_name
    ...
    preprocessors:
      - type: prepare
        mapping:
          user_id: source.users[next].id
```

##### hcl

```terraform
call "req_name" {
  preprocessor "prepare" {
    mapping = {
      user_id = "source.users[next].id"
    }
  }
}
```

#### Postprocessors

##### assert/response

Checks header and body content

Upon assertion, further scenario execution is dropped

```terraform
postprocessor "assert/response" {
  payload = ["token"]
  status_code = 200
}
```

### Scenarios

This section repeats the same [scenario in HTTP generator](./scenario-http-generator.md#scenarios)

The minimum fields for the script are name and list of requests

```terraform
scenario "scenario_name" {
  requests = [
    "list_call",
    "order_call",
    "order_call",
    "order_call"
  ]
}
```

More - [scenario in HTTP generator](./scenario-http-generator.md#scenarios)


### Sources

Follow - [Variable sources](scenario/variable_source.md)

# References

- [HTTP generator](http-generator.md)
- Best practices
    - [RPS per instance](best_practices/rps-per-instance.md)
    - [Shared client](best_practices/shared-client.md)

---

[Home](../index.md)
