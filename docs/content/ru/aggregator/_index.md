---
title: Аггрегатор
description:
categories: [Aggregator]
weight: 10
---

Агрегатор собирает замеры запросов и сохраняет их в каком-нибудь месте.


### 1. phout

**Минимальный конфиг**

```yaml
result:
  type: phout
  destination: file_path.log
```

**Весь конфиг**

```yaml
result:
  type: phout
  destination: file_path.log
  id: false    # Print ammo ids if true.
  flush-time: 1s    # Период сброса буфера в файл, строкой с единицей: от 1ms до 1m.
  sample-queue-size: 262144
  buffer-size: 1048576
```

### 2. jsonlines

**Минимальный конфиг**

```yaml
result:
  type: jsonlines
  sink: 
    type: file
    path: file_path
```

**Весь конфиг**

```yaml
result:
  type: jsonlines
  sink: 
    type: file
    path: file_path
  buffer-size: 1048576
  flush-interval: 1s
  sample-queue-size: 131072
  marshal-float-with-6-digits: false
  sort-map-keys: false
```

Какие еще типы для `sink` существуют смотрите [тут](./sink.md)

### 3. json

Эта псевдоним для `jsonlines`

### 4. log

Вывод данных аггрегатора в о стандартный лог Пандоры

```yaml
result:
  type: log
```

### 5. discard

Отказ от вывода аггрегатора

```yaml
result:
  type: discard
```