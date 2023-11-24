[К содержанию](index.md)

---

# Конфигурация

- [Основная конфигурация](#basic-configuration)
- [Мониторинг и логирование](#monitoring-and-logging)
- [Переменные из переменных окружения и файлов](#variables-from-env-and-files)

## Основная конфигурация

Pandora поддерживает файлы конфигурации в формате `YAML`. Создайте новый файл с именем `load.yaml` и добавьте
в него следующие строки:

```yaml
pools:
  - id: HTTP pool                    # pool name (for your choice)
    gun:
      type: http                     # gun type
      target: example.com:80         # gun target
    ammo:
      type: uri                      # ammo format
      file: ./ammo.uri               # ammo File
    result:
      type: phout                    # report format (phout is compatible with Yandex.Tank)
      destination: ./phout.log       # report file name

    rps:                             # shooting schedule
      type: line                     # linear growth
      from: 1                        # from 1 response per second
      to: 5                          # to 5 responses per second
      duration: 60s                  # for 60 seconds

    startup:                         # instances startup schedule
      type: once                     # start 10 instances
      times: 10
```

## Мониторинг и логирование

Вы можете включить отладочную информацию (мониторинг и профилирование).

```yaml
log:                                 # gun logging configuration
  level: error                       # log only `error` messages (`debug` for verbose logging)

monitoring:
  expvar:                            # gun statistics HTTP server
    enabled: true
    port: 1234
  cpuprofile:                        # cpu profiling
    enabled: true
    file: "cpuprofile.log"
  memprofile:                        # mem profiling
    enabled: true
    file: "memprofile.log"
```


## Переменные из переменных окружения и файлов

В конфигурации можно использовать переменные из переменных окружения или из файлов.

Используйте шаблон - `${...}`.

Переменные окружения: `${env:MY_ENV}`

Переменные из файлов: `${property:path/file.property#MY_FIELD}`.

Содержимое файла должно быть: `MY_FIELD=data`

Пример:

```yaml
pools:
  - ammo:
    type: uri
    file: ./ammofile
    headers:
      - "[Host: yourhost.tld]"
      - "[User-Agent: ${env:LOAD_USER_AGENT}]"
      - "[Custom-Header: ${property:path/file.property#MY_FIELD}]"
```

Переменные можно использовать не только в секции `headers`, но и в любых других полях конфигурации.

---

[К содержанию](index.md)
