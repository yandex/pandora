[Домой](../index.md)

---

# Источники переменных

Используются в

- [Сценарный генератор / HTTP](../scenario-http-generator.md)
- [Сценарный генератор / gRPC](../scenario-grpc-generator.md)

Источники переменных

## csv file

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

## json file

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

## variables

Пример

```terraform
variable_source "global" "variables" {
  variables = {
    host = localhost
    port = 8090
  }
}
```

Создание источника с переменными. Добавление ему имени `global`.

Использование переменных из данного источника

```gotempate
{% raw %}{{.source.global.host}}:{{.source.global.port}}{% endraw %}
```

Дополнительная особенность данного источника - возможность использовать функции рандомизации

Подробнее см [функции рандомизации](functions.md)

---

- [Сценарный генератор / HTTP](../scenario-http-generator.md)
- [Сценарный генератор / gRPC](../scenario-grpc-generator.md)

---

[Домой](../index.md)
