[К содержанию](index.md)

---

# Первый тест

You can use Pandora alone or use it with [Yandex.Tank](https://yandextank.readthedocs.io/en/latest/core_and_modules.html#pandora) as a test runner and [Overload](https://overload.yandex.net) as a result viewer. In the second case Pandora's configuration is the same, but you will embed it into Yandex.Tank's config.

## Config file

Pandora supports config files in [YAML](https://en.wikipedia.org/wiki/YAML) format. Create a new file named `load.yaml` and add following lines in your favourite editor:

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

Visit [Configuration](config.md) page for more details.


Run your tests:

```
pandora load.yaml
```

The results are in `phout.log`. Use [Yandex.Tank](https://yandextank.readthedocs.io/en/latest/core_and_modules.html#pandora) and [Overload](https://overload.yandex.net) to plot them.

---

[К содержанию](index.md)
