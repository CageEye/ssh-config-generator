# Description

This tool will add all of the PROD servers to your local `~.ssh/config` file.

## Requirements

- This tool is supported only for Linux and MacOS systems. For use on Windows look into the WSL (Windows Subsystem for Linux)

- `go` must be installed on to build a binary on a system.

- `make` should be installed for an easy setup.

## Installation

To build, and setup this tool run:

```bash
make
```

## Usage

To update ssh list run:
```bash
cageeyessh
```

To connect to a farm or a cpu server you can now use the `Host` from the `~/.ssh/prod/` files. For example:

```bash
ssh cermaq-ytre-koven-farm-cage-m3
```

You can use auto-completion feature by pressing the `<Tab>` key.

Cages that doesn't have a **cage name** can be identified by the **CPU unit ID**:

```bash
ssh cageeye-terrak-test-rig-farm-cage-16-02-63
```

