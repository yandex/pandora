[Home](../index.md)

---

# Installation

[Download](https://github.com/yandex/pandora/releases) binary release or build from source.

Pandora uses **go modules**.

```bash
git clone https://github.com/yandex/pandora.git
cd pandora
go mod download
```

You can also cross-compile for other arch/os:

```bash
GOOS=linux GOARCH=amd64 go build
```

---

[Home](../index.md)
