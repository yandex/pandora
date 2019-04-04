Ammo providers
==============

Ammo provider is a source of test data: it makes ammo object.

There is a common rule for any (built-in) provider: data supplied by ammo provider are records that will be pushed via established connection to external host (defined in pandora config via `pool.gun.target` option). Thus, you cannot define in the ammofile to which `physical` host your ammo will be sent.


http/json
---------

jsonline format, 1 row — 1 json-encoded ammo.

Pay attention to special header `Host` defined ``outside`` of Headers dictionary.

`Host` inside Headers section will be silently ignored.

Ammofile sample:
::

  {"uri": "/", "method": "GET", "headers": {"Accept": "*/*", "Accept-Encoding": "gzip, deflate", "User-Agent": "Pandora"}, "host": "example.com"}

Config sample:

.. code-block:: yaml

  pools:
    - ammo:
        type: http/json                # ammo format
        file: ./ammofile               # ammo file path


raw (request-style)
-------------------

Raw HTTP request format. If you like to use `telnet` firing HTTP requests, you'll love this.
Also known as phantom's `request-style`.

File contains size-prefixed HTTP requests. Each ammo is prefixed with a header line (delimited with \n), which consists of
two fields delimited by a space: ammo size and tag. Ammo size is in bytes (integer, including special characters like CR, LF).
Tag is a string.
You can read about this format (with detailed instructions) at
`Yandex.Tank documentation <https://yandextank.readthedocs.io/en/latest/tutorial.html#request-style>`_

Ammofile sample:
::

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


Config sample:

.. code-block:: yaml

  pools:
    - ammo:
        type: raw                      # ammo format
        file: ./ammofile               # ammo file path

You can redefine any headers using special config option `headers`. Format: list of strings.

Example:

.. code-block:: yaml

  pools:
    - ammo:
        type: raw                      # ammo format
        file: ./ammofile               # ammo file path
        headers:
          - "[Host: yourhost.tld]"
          - "[User-Agent: some user agent]"

uri-style
---------

List of URIs and headers

Ammofile sample:
::

  [Connection: close]
  [Host: your.host.tld]
  [Cookie: None]
  /?drg tag1
  /
  /buy tag2
  [Cookie: test]
  /buy/?rt=0&station_to=7&station_from=9

Config sample:

.. code-block:: yaml

  pools:
    - ammo:
        type: uri                      # ammo format
        file: ./ammofile               # ammo file path


You can redefine any headers using special config option `headers`. Format: list of strings.

Example:

.. code-block:: yaml

  pools:
    - ammo:
        type: uri                      # ammo format
        file: ./ammofile               # ammo file path
        headers:
          - "[Host: yourhost.tld]"
          - "[User-Agent: some user agent]"
