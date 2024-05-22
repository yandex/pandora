[Домой](index.md)

---

# gRPC генератор

Полный конфиг grpc генератора

```yaml
gun:
  type: grpc
  target: '[hostname]:443'
  timeout: 15s              # Таймаут для запросов gRPC. По умолчанию: 15s
  tls: false                # Если true, Pandora принимает любой сертификат, представленный сервером, и любое имя хоста в этом сертификате. По умолчанию: false
  reflect_port: 8000        # Если ваш рефлекшн сервис находится на отличном от основного сервера порту
  reflect_metadata:         # Отдельные metadata данные для рефлекшн сервис
    auth: Token
  shared-client:
    enabled: true           # Если TRUE, генератор будет использовать общий транспортный клиент для всех инстансов
    client-number: 1        # Количество общих клиентов можно увеличить. По умолчанию 1
  dial_options:
    authority: some.host    # Указывает значение, которое будет использоваться в качестве псевдозаголовка :authority и имени сервера в процессе аутентификации.
    timeout: 1s             # Таймаут установки gRPC соединения. По умолчанию: 1s
  answlog:
    enabled: true
    path: ./answ.log
    filter: all            # all - все http-коды, warning - логировать 4xx и 5xx, error - логировать только 5xx. По умолчанию: error
```

## Маппинг кодов ответа

В качестве клиента Пандора использует gRPC клиент от google.golang.org/grpc (https://github.com/grpc/grpc-go)

Но для унификации графиков преобразует их в HTTP коды.

### Таблица маппинга gPRC StatusCode -> HTTP StatusCode

| gRPC Status Name   | gRPC Status Code | HTTP Status Code |
|--------------------|------------------|------------------|
| OK                 | 0                | 200              |
| Canceled           | 1                | 499              |
| InvalidArgument    | 3                | 400              |
| DeadlineExceeded   | 4                | 504              |
| NotFound           | 5                | 404              |
| AlreadyExists      | 6                | 409              |
| PermissionDenied   | 7                | 403              |
| ResourceExhausted  | 8                | 429              |
| FailedPrecondition | 9                | 400              |
| Aborted            | 10               | 409              |
| OutOfRange         | 11               | 400              |
| Unimplemented      | 12               | 501              |
| Unavailable 14     | 14               | 503              |
| Unauthenticated 16 | 16               | 401              |
| unknown            | -                | 500              |


# Смотри так же

- [Сценарный генератор / gRPC](scenario-grpc-generator.md)
- Практики использования
  - [RPS на инстанс](best_practices/rps-per-instance.md)
  - [Общий транспорт](best_practices/shared-client.md)

---

[Домой](index.md)
