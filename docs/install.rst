Installation
============

`Download <https://github.com/yandex/pandora/releases>`_ binary release or build from source.

We use go 1.11 modules.
If you build pandora inside $GOPATH, please make sure you have env variable `GO111MODULE` set to `on`.

.. code-block:: bash

  git clone https://github.com/yandex/pandora.git
  make deps
  go install



You can also cross-compile for other arch/os:

.. code-block:: bash

  GOOS=linux GOARCH=amd64 go build
