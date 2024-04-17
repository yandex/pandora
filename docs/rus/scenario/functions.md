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

`{{ uuid }}`

### randInt

`{{ randInt }}`

`{{ randInt 10 }}`

`{{ randInt 100 200 }}`

`{{ randInt 200 .source.global.max_rand_int }}` 

### randString

`{{ randString }}`

`{{ randString 10 }}`

`{{ randString 10 abcde }}`

`{{ randString 20 .source.global.letters }}`

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


---

- [Сценарный генератор / HTTP](../scenario-http-generator.md)
- [Сценарный генератор / gRPC](../scenario-grpc-generator.md)

---

[Домой](../index.md)
