Configuration
===============

Basic configuration
-------------------

Pandora supports config files in `YAML`_ format. Create a new file named ``load.yaml`` and add following lines in your favourite editor:

.. code-block:: yaml

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


Monitoring and Logging
----------------------

You can enable debug information about gun (e.g. monitoring and additional logging).

.. code-block:: yaml

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


Variables from env and files
----------------------------

You can use variables in the config from environment variables or from files.

The template format is ``${...}``.

Environment variable ``${env:MY_ENV}``
Variable from file ``${property:path/file.property#MY_FIELD}``.

The contents of the file must be

``MY_FIELD=data``

Example config

.. code-block:: yaml
  pools:
    - ammo:
        type: uri
        file: ./ammofile
        headers:
          - "[Host: yourhost.tld]"
          - "[User-Agent: ${env:LOAD_USER_AGENT}]"
          - "[Custom-Header: ${property:path/file.property#MY_FIELD}]"

You can use variables not only in the header section but also in other configuration fields.

