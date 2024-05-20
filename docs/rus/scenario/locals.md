[Домой](../index.md)

---

#  Locals

Предисловие: 

Формирование сценария из hcl состоит из 2-х этапов:
- этап парсинга - преобразовывает hcl файл в внутреннюю структуру генератора
- этап выполнения - выполнение шагов:
  - препроцессинга
  - шаблонизации
  - постпроцессинга

В данной статье рассматривается этап парсинга.

## HCL

Так же как и в terraform вы можете использовать блок `locals`, который вы можете использовать для создания 
дополнительных переменных. Важно, что данные переменные будут использоваться только при парсинге HCL и их нельзя 
использовать на этапе выполнения.

### Функции

Вы можете использовать [функции HCL](functions.md#функции-hcl)

### Пример использования

Вы можете использовать переменные locals для определения общих заголовков.

Обратите внимание на использование функции `merge()`

```hcl
locals {
  common_headers = {
    Content-Type  = "application/json"
    Useragent     = "Yandex"
  }
  next = "next"
}
locals {
  // смержим новую переменную с локальной переменной local.common_headers
  auth_headers = merge(local.common_headers, {
    Authorization = "Bearer {{.request.auth_req.postprocessor.token}}"
  })
  next = "next"
}

request "list_req" {
  // смержим новую переменную с локальной переменной local.common_headers
  method = "GET"
  headers = merge(local.common_headers, {
    Authorization = "Bearer {{.request.auth_req.postprocessor.token}}"
  })
  tag = "list"
  uri = "/list"

  postprocessor "var/jsonpath" {
    mapping = {
      item_id = "$.items[0]"
      items   = "$.items"
    }
  }
}
```

## YAML

В yaml формате вы можете использовать якоря.

Для общих переменных можно использовать вспомогательный блок locals

```yaml
locals:
  global-headers: &global-headers
    Content-Type: application/json
    Useragent: Yandex

requests:
  - name: auth_req
    headers:
      <<: *global-headers
  - name: list_req
    headers: &auth-headers
      <<: *global-headers
      Authorization: Bearer {{.request.auth_req.postprocessor.token}}
  - name: order_req
    headers:
      <<: *auth-headers
```

---

- [Сценарный генератор / HTTP](../scenario-http-generator.md)
- [Сценарный генератор / gRPC](../scenario-grpc-generator.md)

---

[Домой](../index.md)
