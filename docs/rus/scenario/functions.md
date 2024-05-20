[Домой](../index.md)

---

# Функции рандомизации

Вы можете использовать функции для генерации рандомных значений

- randInt
- randString
- uuid

Использовать их можно в разных частях сценариев с некоторыми особенностями использования 

- [В шаблона](#в-шаблонах)
- [В источник данных - variables](#в-источнике-данных---variables)
- [В препроцессорах](#в-препроцессорах)

## Использование

### uuid

Генерирует случайных uuid v4

### randInt

Генерирует псевдослучайное значение

Аргументы не обязательны. При вызове функции без аргументов будет сгенерировано случайное число в диапазоне 0-9

Передать 1 аргумент - будет сгенерировано случайное число в диапазоне от 0 до этого числа

Передать 2 аргумента - будет сгенерировано случайное число в диапазоне между этими числами

### randString

Генерирует случайную строку

Аргументы не обязательны. При вызове функции без аргументов будет сгенерирован 1 случайный символ

Передать 1 аргумент (число) X - будет сгенерирована строка длиной X

Передать 2 аргумент (строка символов) Y - для генерации будут использьваны только символы из указанной строки Y

## В шаблонах

Так как используется стандартные шаблонизатор Го в нем можно использовать встроенные функции
https://pkg.go.dev/text/template#hdr-Functions

### uuid

```gotemplate
{% raw %}{{ uuid }}{% endraw %}
```

### randInt

без аргументов
```gotemplate
{% raw %}{{ randInt }}{% endraw %}
```

1 аргумент
```gotemplate
{% raw %}{{ randInt 10 }}{% endraw %}
```

2 аргумента
```gotemplate
{% raw %}{{ randInt 100 200 }}{% endraw %}
```

2 аргумента используем из источника переменных
```gotemplate
{% raw %}{{ randInt 200 .source.global.max_rand_int }}{% endraw %} 
```

### randString

без аргументов
```gotemplate
{% raw %}{{ randString }}{% endraw %}
```

1 аргумент
```gotemplate
{% raw %}{{ randString 10 }}{% endraw %}
```

2 аргумента
```gotemplate
{% raw %}{{ randString 10 abcde }}{% endraw %}
```

2 аргумента используем из источника переменных
```gotemplate
{% raw %}{{ randString 20 .source.global.letters }}{% endraw %}
```

## В источник данных - variables

Вы можете использовать функции генерации рандомный значений в источнике переменных типа `variables`

Вызов функций необходимо передавать в виде строки (в кавычках)

```terraform
variable_source "global" "variables" {
  variables = {
    my_uuid = "uuid()"
    my_random_int1 = "randInt()"                # без аргументов
    my_random_int2 = "randInt(10)"              # 1 аргумент
    my_random_int3 = "randInt(100, 200)"        # 2 аргумента
    my_random_string1 = "randString()"          # без аргументов
    my_random_string2 = "randString(10)"        # 1 аргумент
    my_random_string3 = "randString(100, abcde)" # 2 аргумента
  }
}
```

## В препроцессорах

Вы можете использовать функции генерации рандомный значений в препроцессорах

```terraform
preprocessor {
  mapping = {
    my_uuid = "uuid()"
    my_random_int1 = "randInt()"                # без аргументов
    my_random_int2 = "randInt(10)"              # 1 аргумент
    my_random_int3 = "randInt(100, 200)"        # 2 аргумента
    my_random_int4 = "randInt(100, .request.my_req_name.postprocessor.var_from_response)" # 2 аргумента используем из ответа запроса my_req_name
    my_random_string1 = "randString()"          # без аргументов
    my_random_string2 = "randString(10)"        # 1 аргумент
    my_random_string3 = "randString(100, abcde)" # 2 аргумента
    my_random_string4 = "randString(100, .request.my_req_name.postprocessor.var_from_response)"  # 2 аргумента используем из ответа запроса my_req_name
  }
}
```

# Функции HCL

При парсинге HCL доступны следующие функции

- [coalesce](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/coalesce)
- [coalescelist](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/coalescelist)
- [compact](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/compact)
- [concat](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/concat)
- [distinct](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/distinct)
- [element](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/element)
- [flatten](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/flatten)
- [index](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/index-fn)
- [keys](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/keys)
- [lookup](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/lookup)
- [merge](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/merge)
- [reverse](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/reverse)
- [slice](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/slice)
- [sort](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/sort)
- [split](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/string/split)
- [values](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/values)
- [zipmap](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/zipmap)

---

- [Сценарный генератор / HTTP](../scenario-http-generator.md)
- [Сценарный генератор / gRPC](../scenario-grpc-generator.md)

---

[Домой](../index.md)
