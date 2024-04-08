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
  - id: HTTP pool                    # идентификатор инстанс пула
    gun:
      type: http                     # тип генератора
      target: example.com:80         # ... далее идут настройки генератора. Зависят от его типа
    ammo:
      type: uri                      # тип провайдера
      file: ./ammo.uri               # ... далее идут настройки провайдера. Зависят от его типа
    result:
      type: phout                    # тип агрегатора (phout - совместим Yandex.Tank)
      destination: ./phout.log       # report file name

    rps-per-instance: false          # секция rps считается для каждого инстанса или для всего теста. false - для всего теста

    rps:                             # планировщик нагрузки
      type: line                     # тип планировщика
      from: 1                        # ... далее идут настройки планировщика. Зависят от его типа
      to: 5
      duration: 60s

    startup:                         # запуск инстансов
      type: once                     # тип профиля запуска инстансов
      times: 10                      # ... далее идут настройки планировщика. Зависят от его типа
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
