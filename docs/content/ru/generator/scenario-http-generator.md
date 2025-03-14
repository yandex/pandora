---
title: Scenario generator / HTTP
description: Scenario generator / HTTP
categories: [generator]
tags: [scenario, generator, http]
weight: 3
---

## Конфигурация

Вам необходимо использовать генератор и провайдер типа `http/scenario`

```yaml
pools:
  - id: Pool name
    gun:
      type: http/scenario
      target: localhost:80
    ammo:
      type: http/scenario
      file: payload.hcl
```

### Генератор

Минимальная конфигурация генератора выглядит так

```yaml
gun:
  type: http/scenario
  target: localhost:80
```

Так же есть `type: http2/scenario` генератор

```yaml
gun:
  type: http2/scenario
  target: localhost:80
```

Для сценарного генератора поддерживаются все настройки обычного [HTTP генератора](http-generator.md)

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

Сценарий - это последовательность запросов. То есть вам потребуется описать в сценарии какие запросы в каком порядке
должны выполняться.

Запрос - HTTP запрос. Имеет стандартные поля HTTP запроса плюс дополнительные. См [Requests](#requests).

### HCL пример

```terraform
locals {
  common_headers = {
    Content-Type  = "application/json"
    Useragent     = "Yandex"
  }
  next = "next"
}
locals {
  auth_headers = merge(local.common_headers, {
    Authorization = "Bearer {{.request.auth_req.postprocessor.token}}"
  })
  next = "next"
}
variable_source "source_name" "file/csv" {
  file              = "file.csv"
  fields            = ["id", "name"]
  ignore_first_line = true
  delimiter         = ","
}

request "request_name" {
  method  = "POST"
  uri     = "/uri"
  headers = merge(local.common_headers, {
    Authorization = "Bearer {{.request.auth_req.postprocessor.token}}"
  })
  tag       = "tag"
  body      = <<EOF
<body/>
EOF

  templater {
    type = "text"
  }

  preprocessor {
    mapping = {
      new_var = "source.var_name[next].0"
    }
  }
  postprocessor "var/jsonpath" {
    mapping = {
      new_var = "$.auth_key"
    }
  }
}


scenario "scenario_name" {
  weight           = 1
  min_waiting_time = 1000
  requests         = [
    "request_name",
  ]
}
```

Так же пример можно посмотреть в тестах https://github.com/yandex/pandora/blob/dev/tests/grpc_scenario/testdata/grpc_payload.hcl


### YAML пример

```yaml
locals:
  my-headers: &global-headers
    Content-Type: application/json
    Useragent: Yandex
variable_sources:
  - type: "file/csv"
    name: "source_name"
    ignore_first_line: true
    delimiter: ","
    file: "file.csv"
    fields: [ "id", "name" ]

requests:
  - name: "request_name"
    uri: '/uri'
    method: POST
    headers:
      <<: *global-headers
    tag: tag
    body: '<body/>'
    preprocessor:
      mapping:
        new_var: source.var_name[next].0
    templater:
      type: text
    postprocessors:
      - type: var/jsonpath
        mapping:
          token: "$.auth_key"

scenarios:
  - name: scenario_name
    weight: 1
    min_waiting_time: 1000
    requests: [
      request_name
    ]
```

### Locals

Про блок locals смотрите в отдельной [статье Locals](scenario/locals.md)

## Возможности

### Запросы

Поля

- method
- uri
- headers
- body
- **name**
- tag
- templater
- preprocessors
- postprocessors

#### Шаблонизатор

Поля `uri`, `headers`, `body` шаблонризируются.

Используется стандартный go template.

##### Имена переменных в шаблонах

Имена переменных имеют полный путь их определения.

Например

Переменная `users` из источника `user_file` - `{% raw %}{{.source.user_file.users}}{% endraw %}`

Переменная `item` из препроцессора запроса `list_req` - `{% raw %}{{.request.list_req.preprocessor.item}}{% endraw %}`

Переменная `token` из постпроцессора запроса `list_req` - `{% raw %}{{.request.list_req.postprocessor.token}}{% endraw %}`

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

Используется для нового маппинга переменных

У препроцессора есть возможность работать с массивами с помощью модификаторов

- next
- last
- rand

##### yaml

```yaml
requests:
  - name: req_name
    ...
    preprocessor:
      mapping:
        user_id: source.users[next].id
```

##### hcl

```terraform
request "req_name" {
  preprocessor {
    mapping = {
      user_id = "source.users[next].id"
    }
  }
}
```

Так же в препроцессоре есть возможность создавать переменные с использованием функций рандомизации
- randInt()
- randString()
- uuid()

Подробнее про функции рандомизации см в [документации](scenario/functions.md)

#### Postprocessors

##### var/jsonpath

Пример hcl

```terraform
request "your_request_name" {
  postprocessor "var/jsonpath" {
    mapping = {
      token = "$.auth_key"
    }
  }
}
```

##### var/xpath

```terraform
request "your_request_name" {
  postprocessor "var/xpath" {
    mapping = {
      data = "//div[@class='data']"
    }
  }
}
```

##### var/header

Создает новую переменную из заголовков ответа.

Есть возможность через pipe указывать простейшие строковые манипуляции.

Модификаторы:
- lower
- upper
- substr(from, length) - где `length` - опционально
- replace(search, replace)

Модификаторы можно выстраивать в цепочку.

**Пример:**

Если вам приходит ответ с заголовками
```http request
X-Trace-ID: we1fswe284awsfewf
Authorization: Basic Ym9zY236Ym9zY28= 
```

И вам требуется сохранить для дальнейшего использования `traceID=we1fswe284awsfewf` & `auth=ym9zy236ym9zy28`
вы можете использовать постпроцессор с модификаторами

```terraform
request "your_request_name" {
  postprocessor "var/header" {
    mapping = {
      traceID = "X-Trace-ID"
      auth    = "Authorization|lower|replace(=,)|substr(6)"
    }
  }
}
```

В шаблонах вы можете использовать результать данного постпроцессора как
```gotemplate
`{% raw %}{{.request.your_request_name.postprocessor.auth}}{% endraw %}`
`{% raw %}{{.request.your_request_name.postprocessor.traceID}}{% endraw %}`
```

##### assert/response

Проверяет значения заголовков и тела

Если матчинг не срабатывает, прекращает дальнейшее выполнение сценария

```terraform
request "your_request_name" {
  postprocessor "assert/response" {
    headers = {
      "Content-Type" = "application/json"
    }
    body        = ["token"]
    status_code = 200

    size {
      val = 10000
      op  = ">"
    }
  }
}
```

##### assert/json-schema

Проверяет соответствие ответа от сервера указанной json-схеме

**HCL-примеры конфигурации:**

**Использование схемы из файла:**

```terraform
request "your_request_name" {
  postprocessor "assert/json-schema" {
    source = "file"
    schema = "my_json_schema.json"
  }
}
```
**Использование схемы из json-строки:**

```terraform
request "your_request_name" {
  postprocessor "assert/json-schema" {
    source = "json_string"
    schema = "{ \"type\" : \"object\" }"
  }
}
```

**Использование схемы, которая расположена на хосте:**

**Важно: обязательно указывайте схему (http/https)**

```terraform
request "your_request_name" {
  postprocessor "assert/json-schema" {
    source = "url"
    schema = "https://examplehost.com/#validation_schema.json"
  }
}
```

**YAML-пример конфигурации**

**Использование схемы из файла:**
```yaml
requests:
  - name: "your_request_name"
    postprocessors:
      - type: assert/json-schema
        source: file
        schema: "my_json_schema.json"
```

**Использование схемы из json-строки:**
```yaml
requests:
  - name: "your_request_name"
    postprocessors:
      - type: assert/json-schema
        source: json_string
        schema: '{ "type" : "array" }'
```

**Использование схемы, которая расположена на хосте:**

**Важно: обязательно указывайте схему (http/https)**
```yaml
requests:
  - name: "your_request_name"
    postprocessors:
      - type: assert/json-schema
        source: url
        schema: "https://examplehost.com/#validation_schema.json"
```

### Scenarios

Минимальные поля для сценария - имя и перечень запросов

```terraform
scenario "scenario_name" {
  requests = [
    "list_req",
    "order_req",
    "order_req",
    "order_req"
  ]
}
```

Можно указать мултипликатор повторения запросов

```terraform
scenario "scenario_name" {
  requests = [
    "list_req",
    "order_req(3)"
  ]
}
```

Можно указать задержку sleep(). Параметр в миллисекундах

```terraform
scenario "scenario_name" {
  requests = [
    "list_req",
    "sleep(100)",
    "order_req(3)"
  ]
}
```

Вторым аргументом в запросы указывается sleep для запросов с мултипликаторами

```terraform
scenario "scenario_name" {
  requests = [
    "list_req",
    "sleep(100)",
    "order_req(3, 100)"
  ]
}
```

Параметр `min_waiting_time` описывает минимальное время выполнения сценария. То есть будет добавлен sleep в конце всего
сценария, если сценарий выполнится быстрее этого параметра.

```terraform
scenario "scenario_name" {
  min_waiting_time = 1000
  requests         = [
    "list_req",
    "sleep(100)",
    "order_req(3, 100)"
  ]
}
```

В одном файле можно описывать множество сценариев

Параметр `weight` - вес распределения каждого сценария. Чем больше вес, тем чаще будет выполняться сценарий.


```terraform
scenario "scenario_first" {
  weight   = 1
  requests = [
    "auth_req(1, 100)",
    "list_req(1, 100)",
    "order_req(3, 100)"
  ]
}

scenario "scenario_second" {
  weight   = 50
  requests = [
    "mainpage",
  ]
}

```

### Sources

См документ - [Источники переменных](scenario/variable_source.md)

# Смотри так же

- [HTTP генератор](http-generator.md)
- [Практики использования](best-practices)

