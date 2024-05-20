[Домой](index.md)

---

# Сценарный генератор / gRPC

- [Конфигурация](#конфигурация)
    - [Генератор](#генератор)
    - [Провайдер](#провайдер)
- [Описание формата сценариев](#описание-формата-сценариев)
    - [Общий принцип](#общий-принцип)
    - [HCL пример](#hcl-пример)
    - [YAML пример](#yaml-пример)
    - [Locals](#locals)
- [Возможности](#возможности)
    - [Вызовы](#вызовы)
        - [Шаблонизатор](#шаблонизатор)
            - [Имена переменных в шаблонрах](#имена-переменных-в-шаблонах)
            - [Функции в шаблонах](#функции-в-шаблонах)
        - [Preprocessors](#preprocessors)
            - [prepare](#prepare)
        - [Postprocessors](#postprocessors)
            - [assert/response](#assertresponse)
    - [Scenarios](#scenarios)
    - [Sources](#sources)
- [Смотри так же](#cмотри-так-же)

## Конфигурация

Вам необходимо использовать генератор и провайдер типа `grpc/scenario`

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

### Генератор

Минимальная конфигурация генератора выглядит так

```yaml
gun:
  type: grpc/scenario
  target: localhost:8888
```

Для сценарного генератора поддерживаются все настройки обычного [gRPC генератора](grpc-generator.md)

### Провайдер

Провайдер принимает всего один параметр - путь до файла с описанием сценария

```yaml
ammo:
  type: http/scenario
  file: payload.hcl
```

Поддерживает файлы расширений

- hcl
- yaml
- json

## Описание формата сценариев

Поддерживает форматы

- hcl
- yaml
- json

### Общий принцип

В одном файле можно описывать несколько сценариев. У сценария есть имя по которому один сценарий отличается от другого.

Сценарий - это последовательность gRPC вызовов. То есть вам потребуется описать в сценарии какие вызовы в каком порядке
должны выполняться.

Вызов - gRPC вызов. Имеет стандартные поля gRPC вызова плюс дополнительные. См [Calls](#calls).

### HCL пример

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

Так же пример можно посмотреть в тестах https://github.com/yandex/pandora/blob/dev/tests/grpc_scenario/testdata/grpc_payload.hcl


### YAML пример

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

## Возможности

### Вызовы

Поля

- call
- tag
- metadata
- preprocessors
- payload
- postprocessors

### Шаблонизатор

Поля `metadata`, `payload` шаблонризируются.

Используется стандартный go template.

#### Имена переменных в шаблонах

Имена переменных имеют полный путь их определения.

Например

Переменная `users` из источника `user_file` - `{% raw %}{{.source.user_file.users}}{% endraw %}`

Переменная `item` из препроцессора запроса `list_req` - `{% raw %}{{.request.list_req.preprocessor.item}}{% endraw %}`

> Обратите внимание
> Для сохранения подобия с http сценариями секция ответов от grpc вызова сохраняется в раздел `postprocessor`

Переменная `token` из вызова `list_req` - `{% raw %}{{.request.list_req.postprocessor.token}}{% endraw %}`

##### Функции в шаблонах

Так как используется стандартные шаблонизатор Го в нем можно использовать встроенные функции
https://pkg.go.dev/text/template#hdr-Functions

А так же некоторые функции

- randInt
- randString
- uuid

Подробнее про функции рандомизации см в [документации](scenario/functions.md)

#### Preprocessors

Препроцессор - действия выполняются перед шаблонизацией

##### prepare

Используется для нового маппинга переменных

У препроцессора есть возможность работать с массивами с помощью модификаторов

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

Проверяет значения заголовков и тела

Если матчинг не срабатывает, прекращает дальнейшее выполнение сценария

```terraform
postprocessor "assert/response" {
  payload = ["token"]
  status_code = 200
}
```

### Scenarios

Данная секция повторяет такую же [секцию сценариев в HTTP генераторе](./scenario-http-generator.md#scenarios)

Минимальные поля для сценария - имя и перечень запросов

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

Подробнее см [секцию сценариев в HTTP генераторе](./scenario-http-generator.md#scenarios)


### Sources

См документ - [Источники переменных](scenario/variable_source.md)

# Смотри так же

- [HTTP генератор](http-generator.md)
- Практики использования
    - [RPS на инстанс](best_practices/rps-per-instance.md)
    - [Общий транспорт](best_practices/shared-client.md)


---

[Домой](index.md)
