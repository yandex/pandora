[Home](../index.md)

---

# Scenario generator / HTTP

- [Configuration](#configuration)
    - [Генератор](#генератор)
    - [Провайдер](#провайдер)
- [Описание формата сценариев](#описание-формата-сценариев)
    - [Описание формата в HCL](#описание-формата-в-hcl)
    - [Пример в YAML](#пример-в-yaml)
- [Возможности](#возможности)
    - [Requests](#requests)
        - [Шаблонизатор](#шаблонизатор)
            - [Имена переменных в шаблонрах](#имена-переменных-в-шаблонах)
        - [Preprocessors](#preprocessors)
        - [Postprocessors](#postprocessors)
            - [var/jsonpath](#varjsonpath)
            - [var/xpath](#varxpath)
            - [var/header](#varheader)
            - [assert/response](#assertresponse)
    - [Scenarios](#scenarios)
    - [Sources](#sources)
        - [csv file](#csv-file)
        - [json file](#json-file)
        - [variables](#variables)

## Configuration

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

Запрос - HTTP запрос. То есть имеет все стандартные поля HTTP запроса. И дополнительные для работы в сценарии

### Описание формата в HCL

```terraform
variable_source "source_name" "file/csv" {
  file              = "file.csv"
  fields            = ["id", "name"]
  ignore_first_line = true
  delimiter         = ","
}

request "request_name" {
  method  = "POST"
  uri     = "/uri"
  headers = {
    HeaderName = "header value"
  }
  tag       = "tag"
  body      = <<EOF
<body/>
EOF
  templater = "text"

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

### Пример в YAML

```yaml
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
      Header-Name: "header value"
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

## Возможности

### Requests

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

### Шаблонизатор

Поля `uri`, `headers`, `body` шаблонризируются.

Используется стандартный go template.

#### Имена переменных в шаблонах

Имена переменных имеют полный путь их определения.

Например

Переменная `users` из источника `user_file` - `{% raw %}{{.source.user_file.users}}{% endraw %}`

Переменная `token` из постпроцессора запроса `list_req` - `{% raw %}{{.request.list_req.postprocessor.token}}{% endraw %}`

Переменная `item` из препроцессора запроса `list_req` - `{% raw %}{{.request.list_req.preprocessor.item}}{% endraw %}`

#### Preprocessors

Препроцессор - действия выполняются перед шаблонизацией

Используется для нового маппинга переменных

У препроцессора есть возможность работать с массивами с помощью модификаторов
-

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

#### Postprocessors

##### var/jsonpath

Пример hcl

```terraform
postprocessor "var/jsonpath" {
  mapping = {
    token = "$.auth_key"
  }
}
```

##### var/xpath

```terraform
postprocessor "var/xpath" {
  mapping = {
    data = "//div[@class='data']"
  }
}
```

##### var/header

Создает новую переменную из заголовков ответа

Есть возможность через pipe указывать простейшие строковые манипуляции

- lower
- upper
- substr(from, length)
- replace(search, replace)

```terraform
postprocessor "var/header" {
  mapping = {
    ContentType      = "Content-Type|upper"
    httpAuthorization = "Http-Authorization"
  }
}
```

##### assert/response

Проверяет значения заголовков и тела

Если матчинг не срабатывает, прекращает дальнейшее выполнение сценария

```terraform
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

Источники переменных

#### csv file

Пример

```terraform
variable_source "users" "file/csv" {
  file              = "users.csv"                   # required
  fields            = ["user_id", "name", "pass"]   # optional
  ignore_first_line = true                          # optional
  delimiter         = ","                           # optional
}
```

Создание источника из csv. Добавление ему имени `users`.

Использование переменных из данного источника

```gotempate
{% raw %}{{.source.users[0].user_id}}{% endraw %}
```

Параметр `fields` является необязательным.

Если этого параметра нет, то в качестве имен полей будет использоваться имена в первой строке csv файла,
если `ignore_first_line = false`.

Если `ignore_first_line = true` и отсутствуют поля, то в качестве имен будут использоваться порядковые номер

```gotempate
{% raw %}{{.source.users[0].0}}{% endraw %}
```

#### json file

Пример

```terraform
variable_source "users" "file/json" {
  file = "users.json"     # required
}
```

Создание источника из json файла. Добавление ему имени `users`.

Файл должен содержать любой валидный json. Например:

```json
{
    "data": [
        {
            "id": 1,
            "name": "user1"
        },
        {
            "id": 2,
            "name": "user2"
        }
    ]
}
```

Использование переменных из данного источника

```gotempate
{% raw %}{{.source.users.data[next].id}}{% endraw %}
```

Или пример с массивом

```json
 [
    {
        "id": 1,
        "name": "user1"
    },
    {
        "id": 2,
        "name": "user2"
    }
]
```

Использование переменных из данного источника

```gotempate
{% raw %}{{.source.users[next].id}}{% endraw %}
```

#### variables

Пример

```terraform
variable_source "variables" "variables" {
  variables = {
    host = localhost
    port = 8090
  }
}
```

Создание источника с переменными. Добавление ему имени `variables`.

Использование переменных из данного источника

```gotempate
{% raw %}{{.source.variables.host}}:{{.source.variables.port}}{% endraw %}
```

---

[Home](../index.md)
