---
title: HTTP генератор
description: Настройка http/http2 генератора
categories: [Generator]
tags: [generator, http]
weight: 1
---

Полный конфиг http (http2) генератора

```yaml
gun:
  type: http
  target: '[hostname]:443'
  ssl: true                     # Для type: http2 значение false включает h2c, HTTP/2 без TLS
  response-code: http           # По умолчанию HTTP-статус; grpc — для type: http2
  connect-ssl: false            # If true, Pandora accepts any certificate presented by the server and any host name in that certificate. Default: false
  tls-handshake-timeout: 1s     # Maximum waiting time for a TLS handshake. Default: 1s
  disable-keep-alives: false    # If true, disables HTTP keep-alives. Default: false
  disable-compression: true     # If true, prevents the Transport from requesting compression with an "Accept-Encoding: gzip" request header. Default: true
  max-idle-conns: 0             # Maximum number of idle (keep-alive) connections across all hosts. Zero means no limit. Default: 0
  max-idle-conns-per-host: 2    # Controls the maximum idle (keep-alive) connections to keep per-host. Default: 2
  idle-conn-timeout: 90s        # Maximum amount of time an idle (keep-alive) connection will remain idle before closing itself. Zero means no limit. Default: 90s
  response-header-timeout: 0    # Amount of time to wait for a server's response headers after fully writing the request (including its body, if any). Zero means no timeout. Default: 0
  expect-continue-timeout: 1s   # Amount of time to wait for a server's first response headers after fully writing the request headers if the request has an "Expect: 100-continue" header. Zero means no timeout. Default: 1s
  shared-client:
    enabled: true               # Если TRUE, генератор будет использовать общий транспортный клиент для всех инстансов
    client-number: 1            # Количество общих клиентов можно увеличить. По умолчанию 1
  dial:
    timeout: 1s                 # TCP connect timeout. Default: 3s
    dns-cache: true             # Enable DNS cache, remember remote address on first try, and use it in the future. Default: true
    dual-stack: true            # IPv4 is tried soon if IPv6 appears to be misconfigured and hanging. Default: true
    fallback-delay: 300ms       # The amount of time to wait for IPv6 to succeed before falling back to IPv4. Default 300ms
    keep-alive: 120s            # Interval between keep-alive probes for an active network connection Default: 120s
  answlog:
    enabled: true
    path: ./answ.log
    filter: all             # all - all http codes, warning - log 4xx and 5xx, error - log only 5xx. Default: error
  auto-tag:
    enabled: true
    uri-elements: 2         # URI elements used to autotagging. Default: 2
    no-tag-only: true       # When true, autotagged only ammo that has no tag before. Default: true
  httptrace:
    dump: true              # calculate response bytes
    trace: true             # calculate different request stages: connect time, send time, latency, request bytes
```

## gRPC-ответы через HTTP/2 и h2c

Если патроны уже содержат готовый gRPC-запрос, HTTP/2-генератор может интерпретировать ответ как gRPC без рефлексии и перекодирования запроса:

```yaml
pools:
  - id: grpc-over-h2c
    gun:
      type: http2
      target: localhost:8095
      ssl: false                 # h2c; true — HTTP/2 с TLS
      response-code: grpc
      answlog:
        enabled: true
        path: ./answ.log
        filter: warning
        sampling:
          enabled: false         # Записать каждый подходящий ответ
    ammo:
      type: raw
      file: ./grpc.raw
    result:
      type: phout
      destination: ./phout.log
    rps: {type: const, ops: 10, duration: 30s}
    startup: {type: once, times: 10}
```

`response-code` принимает `http` и `grpc`. Поле необязательно: без него используется `http` и записывается HTTP-статус. Значение `grpc` поддерживается только для `type: http2`. Патрон должен содержать готовый gRPC POST-запрос с нужными headers и телом: эта настройка не формирует protobuf-сообщения и не включает повторы запросов.

В gRPC-режиме генератор берёт `grpc-status` из финальных trailers; для trailers-only ответа он читает статус из завершающего блока headers. Если статуса нет либо `Content-Type` ответа не является gRPC, применяется [стандартное преобразование HTTP-кода в gRPC-код](https://github.com/grpc/grpc/blob/master/doc/http-grpc-status-mapping.md). Затем код записывается в `proto_code` как HTTP-эквивалент по [той же таблице, что у штатного gRPC-генератора](./grpc-generator.md#маппинг-кодов-ответа). Например, `grpc-status: 0` даёт `200`, а `grpc-status: 14` даёт `503`. Исходный HTTP-код и источник gRPC-кода сохраняются в answlog. Если HTTP-код отличается от `200`, он имеет приоритет над `grpc-status`: HTTP `503` с `grpc-status: 0` не станет успешным выстрелом.

RTT фиксируется при получении финальных HTTP headers, до чтения body. Выстрел попадает в результат после чтения ответа до конца, когда известны trailers. В answlog записываются исходный gRPC-код, декодированный `grpc-message`, `grpc-status-details-bin` (исходная base64-строка и, если он корректен, декодированное `google.rpc.Status`), HTTP-статус, запрос и headers/trailers ответа. `grpc-message` и детали сохраняются в answlog даже без `grpc-status`; при HTTP `200` и gRPC `Content-Type` повреждённая base64-строка деталей считается gRPC `INTERNAL` (`500`), даже если `grpc-status` отсутствует. При ошибке чтения body записывается доступная HTTP-диагностика и причина ошибки. Бинарное тело ответа не буферизуется для лога. Фильтры `warning` и `error` используют исходные gRPC-коды так же, как штатный gRPC-генератор. Бесконечный стрим не даст финального статуса, пока не завершится.

# Смотри так же

- Практики использования
  - [RPS на инстанс](best_practices/rps-per-instance.md)
  - [Общий транспорт](best_practices/shared-client.md)
