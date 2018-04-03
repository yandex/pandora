Your first test
===============

You can use Pandora alone or use it with `Yandex.Tank`_ as a test runner and
`Overload`_ as a result viewer. In the second case Pandora's configuration is the same, but you will embed it into Yandex.Tank's config.

Config file
-----------

Pandora supports config files in `YAML`_ format. Create a new file named ``load.yaml`` and add following lines in your favourite editor:

.. code-block:: yaml

  pools:
    - id: HTTP pool                    # Pool name
      gun:
        type: http                     # Gun type
        target: example.com:80         # Gun target
      ammo:
        type: uri                      # Ammo format                        
        file: ./ammo.uri               # Ammo File
      result:
        type: phout                    # Report format (phout is for Yandex.Tank)
        destination: ./phout.log       # Report file name

      rps:                             # RPS Schedule
        type: periodic                 # shoot periodically
        period: 0.1s                   # ten batches each second
        max: 30                        # thirty batches total
        batch: 2                       # in batches of two shoots

      startup:                         # Startup Schedule
        type: periodic                 # start Instances periodically
        period: 0.5s                   # every 0.5 seconds
        batch: 1                       # one Instance at a time
        max: 5                         # five Instances total

`ammo.uri`:

::

  /my/first/url
  /my/second/url

Run your tests:


.. code-block:: bash

  pandora load.yaml


The results are in ``phout.log``. Use `Yandex.Tank`_
and `Overload`_ to plot them.

References
----------

.. target-notes::

.. _`Overload`: https://overload.yandex.net
.. _`Yandex.Tank`: http://yandextank.readthedocs.org/en/latest/configuration.html#pandora
.. _`YAML`: https://en.wikipedia.org/wiki/YAML