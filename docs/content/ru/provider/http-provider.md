---
title: HTTP провайдер
description: HTTP провайдер это источник тестовых данных, который создает объекты Payload
categories: [Provider]
tags: [provider, http]
weight: 10
---

Провайдер HTTP является источником тестовых данных: он создает объект Payload.

Существует общее правило для любого (встроенного) провайдера: данные, поставляемые провайдером патронов, - это записи, которые будут переданы через
установленное соединение с внешним хостом (задается в конфигурации pandora через опцию _pool.gun.target_). Таким образом, вы не можете
определить в файле payload, на какой _физический_ хост будут отправлены ваши Payload.

## Тестовые данные

### http/json

формат jsonline, 1 строка - 1 патрон в json-кодировке.

Обратите внимание на специальный заголовок _Host_, определенный `вне` словаря Headers.

_Host_ внутри секции Headers будет молча проигнорирован.

Пример содержимого:

```
{"tag": "tag1", "uri": "/", "method": "GET", "headers": {"Accept": "*/*", "Accept-Encoding": "gzip, deflate", "User-Agent": "Pandora"}, "host": "example.com"}
```

Пример конфига:

```yaml
pools:
  - ammo:
      type: http/json                # ammo format
      file: ./ammofile               # ammo file path
```

В случае, если строки в файле с патронами имеют длину более 64 кБайт следует указывать размер буфера для чтения строк с помощью параметра `maxammosize`. Значение указывается в байтах и должно быть больше самой длинной строки. Пример конфига:

```yaml
pools:
  - ammo:
      type: http/json       # ammo format
      file: ./ammofile      # ammo file path
      maxammosize: 1000000  # maximum ammo size
```

### raw (request-style)

Формат Raw HTTP-запроса. Если вы любите использовать _telnet_, обстреливающий HTTP-запросы, вам понравится это.
Также известен как _request-style_ от Phantom.

Файл содержит HTTP-запросы с префиксом размера. Каждый запрос имеет строку заголовка (разделенную \n), которая состоит из
из двух полей, разделенных пробелом: размер патрона и тег. Размер боеприпаса указывается в байтах (целое число, включая специальные символы, такие как
CR, LF). Тег - это строка. Об этом формате (с подробными инструкциями) вы можете прочитать на сайте
[Документация Яндекс.Танка](https://yandextank.readthedocs.io/en/latest/tutorial.html#request-style)

Пример аммофайла:

```
73 good
GET / HTTP/1.0
Host: xxx.tanks.example.com
User-Agent: xxx (shell 1)

77 bad
GET /abra HTTP/1.0
Host: xxx.tanks.example.com
User-Agent: xxx (shell 1)

78 unknown
GET /ab ra HTTP/1.0
Host: xxx.tanks.example.com
User-Agent: xxx (shell 1)
```

Пример конфига:

```yaml
pools:
  - ammo:
      type: raw                      # ammo format
      file: ./ammofile               # ammo file path
```

Вы можете определить общие заголовки с помощью специальной опции конфигурации `headers`. 
Заголовки в ammo-файле имеют приоритет. Формат: список строк.

Пример:

```yaml
pools:
  - ammo:
      type: raw                      # ammo format
      file: ./ammofile               # ammo file path
      headers:
        - "[Host: yourhost.tld]"
        - "[User-Agent: some user agent]"
```

### uri-style

Список URIs и заголовков

Пример содержимого:

```
[Connection: close]
[Host: your.host.tld]
[Cookie: None]
/?drg tag1
/
/buy tag2
[Cookie: test]
/buy/?rt=0&station_to=7&station_from=9
```

Пример конфига:


```yaml
pools:
  - ammo:
      type: uri                      # ammo format
      file: ./ammofile               # ammo file path
```

Вы можете определить общие заголовки с помощью специальной опции конфигурации `headers`. 
Заголовки в ammo-файле имеют приоритет. Формат: список строк.

Пример:

```yaml
pools:
  - ammo:
      type: uri                      # ammo format
      file: ./ammofile               # ammo file path
      headers:
        - "[Host: yourhost.tld]"
        - "[User-Agent: some user agent]"
```

В случае, если строки в файле с патронами имеют длину более 64 кБайт следует указывать размер буфера для чтения строк с помощью параметра `maxammosize`. Значение указывается в байтах и должно быть больше самой длинной строки. Пример конфига:

```yaml
pools:
  - ammo:
      type: uri             # ammo format
      file: ./ammofile      # ammo file path
      maxammosize: 1000000  # maximum ammo size
```

## Возможности

### Фильтры

Каждый провайдер http позволяет выбрать конкретный ammo для вашего теста из файла ammo с настройкой _chosencases_:

```yaml
pools:
  - ammo:
      type: uri                        # ammo format
      chosencases: [ "tag1", "tag2" ]  # use only "tag1" and "tag2" ammo for this test
      file: ./ammofile                 # ammo file path
```

Теги определяются в файлах патронов, как показано ниже:

#### http/json:

```
{"tag": "tag1", "uri": "/",
```

#### raw (request-style):

```
73 tag1
GET / HTTP/1.0
```

#### uri-style:

```
/?drg tag1
/
/buy tag2
```

### HTTP middlewares

Провайдеры HTTP имеют возможность модифицировать HTTP-запрос непосредственно перед выполнением.
Для этого используются Middlewares. Пример Middleware, устанавливающего заголовок Date в запросе.

```yaml
pools:
  - ammo:
      type: uri
      ...
      middlewares:
        - type: header/date
          location: EST
          headerName: Date
```

Список встроенных Middlewares HTTP:

- header/date

Вы можете создавать собственные промежуточные модули. 
Но для этого вам нужно зарегистрировать их в [custom pandora](generator/custom.md)

```go
import (
    "github.com/yandex/pandora/components/providers/http/middleware"
    "github.com/yandex/pandora/components/providers/http/middleware/headerdate"
    httpRegister "github.com/yandex/pandora/components/providers/http/register"
)

httpRegister.HTTPMW("header/date", func (cfg headerdate.Config) (middleware.Middleware, error) {
    return headerdate.NewMiddleware(cfg)
})
```

Подробнее о том, как писать пользовательские пандоры, читайте в [Custom](generator/custom.md).

### Предварительная загрузка HTTP-патронов

Архитектура Pandora рассчитана на высокую производительность. 
Для достижения высокой производительности Pandora подготавливает payload для каждого экземпляра.

Если у вас **большие запросы** и **большое количество экземпляров**, Pandora начинает использовать много памяти.

На этот случай у HTTP-провайдеров есть флаг `preload`. Если он установлен в `true`, провайдер загрузит файл патронов в
память и использовать тело запроса из памяти.

Пример:

```yaml
pools:
  - ammo:
      type: ...
      ...
      preload: true
```
