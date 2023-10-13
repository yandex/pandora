Pandora's performance
=====================

.. warning::

  New Documentation https://yandex.github.io/pandora/

`Alexander Ivanov`_ made some performance tests for the gun itself. Here are the results.

* Server: NGinx, 32 cores, 64G RAM.
* Tank: 32 cores, 128G RAM.
* Network: 1G.

HTTP requests to nginx
----------------------

Static pages with different sizes. Server delays implemented in Lua script, we can
set delay time using ``sleep`` query parameter:

.. code-block:: lua

    server {
        listen          12999      default;
        listen          [::]:12999 default         ipv6only=on;
        server_name     pandora.test.yandex.net;

        location ~* / {

            rewrite_by_lua_block {
                local args = ngx.req.get_uri_args()
                if args['sleep'] then
                                ngx.sleep(args['sleep']/1000)
                end;
            }

            root /etc/nginx/pandora;
            error_page 404 = 404;

            }

            access_log off;
            error_log off;
    }


* **Connection: Close** 23k RPS

.. image:: screenshot/http_connection_close_td.png
    :align: center
    :alt: Connection:Close, response times distribution

* **Connection: Keep-Alive** 95k RPS

.. image:: screenshot/http_keep_alive_td.png
    :align: center
    :alt: Keep-Alive, response times distribution

* **Response size 10kB** maxed out network interface. OK.
* **Response size 100kb** maxed out network interface. OK.
* **POST requests 10kB** maxed out network interface. OK.
* **POST requests 100kB** maxed out network interface. OK.
* **POST requests 1MB** maxed out network interface. OK.

.. image:: screenshot/http_100kb_net.png
    :align: center
    :alt: 100 kb responses, network load


* **50ms server delay** 30k RPS. OK.
* **500ms server delay** 30k RPS, 30k instances. OK.
* **1s server delay** 50k RPS, 50k instances. OK.
* **10s server delay** 5k RPS, 5k instances. OK.

**All good.**

.. image:: screenshot/http_delay_10s_td.png
    :align: center
    :alt: 10s server delay, response times distribution

.. image:: screenshot/http_delay_10s_instances.png
    :align: center
    :alt: 10s server delay, instances count


* **Server fail during test** OK.

.. image:: screenshot/http_srv_fail_q.png
    :align: center
    :alt: server fail emulation, response times quantiles


Custom scenarios
----------------

Custom scenarios performance depends very much of their implementation. In some our
test we saw spikes caused by GC. They can be avoided by reducing allocation size.
It is a good idea to optimize your scenarios.
Go has `a lot <https://github.com/golang/go/wiki/Performance>`_ of tools helping you
to do this.

.. note:: We used JSON-formatted ammo to specify parameters for each scenario run.

* **Small requests** 35k RPS. OK.
* **Some scenario steps with big JSON bodies** 35k RPS. OK.

.. image:: screenshot/scn_cases.png
    :align: center
    :alt: scenario steps


.. _Alexander Ivanov: ival.net@yandex.ru