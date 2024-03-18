[К содержанию](index.md)

---

# Архитектура

- [Схема](#схема)
- [Типы компонентов](#типы-компонентов)
    - [Provider](#provider)
    - [Instances Pool](#instances-pool)
    - [Scheduler](#scheduler)
    - [Instances and Guns](#instances-and-guns)
    - [Aggregator](#aggregator)

## Схема

Код схемы доступен [здесь](../images/architecture.graphml).
Его можно открыть и редактировать в редакторе [YeD](https://www.yworks.com/en/products/yfiles/yed/).

![architectural scheme](../images/architecture.png)

Pandora - это набор компонентов, взаимодействующих друг с другом с помощью Go каналов.

## Типы компонентов

### Provider

Ammo Provider знает, как создать объект Payload из payload файла (ammo file) или другого внешнего ресурса.
И их задача передать Payload Instance'у. См метод `func (p *Provider) Acquire() (core.Ammo, bool)`

### Instance Pool

**Пул инстансов** управляет созданием **инстансов**. Один инстанс можно представить как одного пользователя, который
**последовательно** отправляет запросы на сервер. Все инстансы из одного пула инстансов получают данные от одного
**провайдера**. Время создания инстанса контролируется **планировщиком**. Все инстансы из одного пула инстансов имеют генераторы
одного типа.

### Scheduler

Scheduler controls other events' times by pushing messages to its underlying channel according to the Schedule.
It can control Instances startup times, RPS amount (requests per second) or other processes.

By combining two types of Schedulers, RPS Scheduler and Instance Startup Scheduler, you can simulate different types of
load.
Instace Startup Scheduler controls the level of parallelism and RPS Scheduler controls throughput.

RPS Scheduler can limit throughput of a whole instances pool, i.e. 10 RPS on 10 instances means 10 RPS overall, or
limit throughput of each instance in a pool individually, i.e. 10 RPS on each of 10 instances means 100 RPS overall.

If you set RPS Scheduler to 'unlimited' and then gradually raise the number of Instances in your system by using
Instance
Startup Scheduler, you'll be able to study
the [scalability](http://www.perfdynamics.com/Manifesto/USLscalability.html)
of your service.

If you set Instances count to a big, unchanged value (you can estimate the needed amount by using
[Little's Law](https://en.wikipedia.org/wiki/Little%27s_law)) and then gradually raise the RPS by using RPS Scheduler,
you'll be able to simulate Internet and push your service to its limits.

You can also combine two methods mentioned above.

### Instances and Guns

Instances takes an ammo, waits for a Scheduler tick and then shoots with a Gun it has. Gun is a tool that sends
a request to your service and measures the parameters (time, error codes, etc.) of the response.

### Aggregator

Aggregator collects measured samples and saves them somewhere.

---

[К содержанию](index.md)
