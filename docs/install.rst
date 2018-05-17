Installation
============

`Download <https://github.com/yandex/pandora/releases>`_ binary release or build from source.

We use `dep <https://github.com/golang/dep>`_ for package management. Install it before proceeding. Then build a binary with go tool (use go >= 1.8.3):

.. code-block:: bash

  go get github.com/yandex/pandora
  cd $GOPATH/src/github.com/yandex/pandora
  dep ensure
  go install

You can also cross-compile for other arch/os:

.. code-block:: bash

  GOOS=linux GOARCH=amd64 go build
