---
title: gRPC провайдер
description: gRPC провайдер это источник тестовых данных, который создает объекты Payload
categories: [Provider]
tags: [provider, grpc]
weight: 10
---

Провайдер gRPC является источником тестовых данных: он создает объект Payload.

Существует общее правило для любого (встроенного) провайдера: данные, поставляемые провайдером патронов - это записи, которые будут переданы через установленное соединение с внешним хостом (задается в конфигурации pandora через опцию _pool.gun.target_). Таким образом, вы не можете определить в файле payload, на какой _физический_ хост будут отправлены ваши Payload.

## Тестовые данные

### grpc/json

Формат jsonline, 1 строка - 1 патрон в json-кодировке.

Пример содержимого:

```json
{"tag": "/Add", "call": "api.Adder.Add", "metadata": {"Authorization": "Bearer $YC_TOKEN"}, "payload": {"x": 21, "y": 12}}
{"tag": "/Add", "call": "api.Adder.Add", "metadata": {"Authorization": "Bearer $YC_TOKEN"}, "payload": {"x": 22, "y": 13}}
{"tag": "/Add", "call": "api.Adder.Add", "metadata": {"Authorization": "Bearer $YC_TOKEN"}, "payload": {"x": 23, "y": 14}}
```
Где:
   * `tag` - уcловное обозначение запроса (тег), которое используется в интерфейсе отображения результатов. Позволяет пометить запросы разными тегами для группировки и фильтрации результатов тестов.
   * `call` - сервис и его вызываемый метод.
   * `metadata` - используется для отправки хедеров, например `Authorization`.
   * `payload` - тело запроса.


При конфигурации генератора нагрузки Pandora с помощью yaml-файла необходимо указать тип `grpc/json` в секции `ammo`. Пример конфига:

```yaml
pools:
  - ammo:
      type: grpc/json     # ammo format
      file: ./ammo.json   # ammo file path
```

В случае, если строки в файле с патронами имеют длину более 64 кБайт следует указывать размер буфера для чтения строк с помощью параметра `maxammosize`. Значение указывается в байтах и должно быть больше самой длинной строки. Пример конфига:

```yaml
pools:
  - ammo:
      type: grpc/json       # ammo format
      file: ./ammo.json     # ammo file path
      maxammosize: 1000000  # maximum ammo size
```

## Возможности

### Фильтры

Каждый провайдер grpc позволяет выбрать конкретный ammo для вашего теста из файла ammo с настройкой `chosencases`:

```yaml
pools:
  - ammo:
      type: grpc/json                  # ammo format
      chosencases: [ "tag1", "tag2" ]  # use only "tag1" and "tag2" ammo for this test
      file: ./ammofile                 # ammo file path
```

### Ограничение числа патронов

Все провайдеры по умолчанию при достижении конца списка патронов начинают чтение с начала.
Для ограничения количества повторов используется параметр `passes`.

Если нужно задать общее ограничение на количество патронов, которые может выдать провайдер, следует использовать параметр `limit`.

Пример конфига:

```yaml
pools:
  - ammo:
      type: grpc/json     # ammo format
      file: ./ammo.json   # ammo file path
      limit: 0            # Limit limits total num of ammo. Unlimited if zero. Default: 0
      passes: 0           # Passes limits ammo file passes. Unlimited if zero. Default: 0
```
