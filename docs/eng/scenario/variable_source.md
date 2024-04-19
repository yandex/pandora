[Home](../../index.md)

---

# Variable sources

Use with:

- [Scenario generator / HTTP](../scenario-http-generator.md)
- [Scenario generator / gRPC](../scenario-grpc-generator.md)


Variable sources

## csv file

Example

```terraform
variable_source "users" "file/csv" {
  file              = "users.csv"                   # required
  fields            = ["user_id", "name", "pass"]   # optional
  ignore_first_line = true                          # optional
  delimiter         = ","                           # optional
}
```

Creating a source from csv. Adding the name `users` to it.

Using variables from this source

```gotempate
{% raw %}{{.source.users[0].user_id}}{% endraw %}
```

The `fields` parameter is optional.

If this parameter is not present, the names in the first line of the csv file will be used as field names,
if `ignore_first_line = false`.

If `ignore_first_line = true` and there are no fields, then ordinal numbers will be used as names

```gotempate
{% raw %}{{.source.users[0].0}}{% endraw %}
```

## json file

Example

```terraform
variable_source "users" "file/json" {
  file = "users.json"     # required
}
```

Creating a source from a json file. Add the name `users` to it.

The file must contain any valid json. For example:

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

Using variables from this source

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

Using variables from this source

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

Creating a source with variables. Add the name `global` to it.

Using variables from this source

```gotempate
{% raw %}{{.source.global.host}}:{{.source.global.port}}{% endraw %}
```

An additional feature of this source is the ability to use randomization functions.

For more details, see [randomization functions](functions.md).

---

- [Scenario generator / HTTP](../scenario-http-generator.md)
- [Scenario generator / gRPC](../scenario-grpc-generator.md)

---

[Home](../../index.md)
