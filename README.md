# lz

A TUI for managing command aliases. Run saved commands with custom interactive argument selections!

![lz demo](lzdemo.gif)

## Installation

### From GitHub Releases

```bash
# Linux amd64
curl -sL https://github.com/joao-alemao/laziest/releases/latest/download/lz_linux_amd64.tar.gz | tar xz
sudo mv lz /usr/local/bin/

# Linux arm64
curl -sL https://github.com/joao-alemao/laziest/releases/latest/download/lz_linux_arm64.tar.gz | tar xz
sudo mv lz /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/joao-alemao/laziest.git
cd laziest
go build -o lz ./cmd/laziest
sudo mv lz /usr/local/bin/
```

## Quick Start

```bash
# 1. Set up shell integration (one-time)
lz init
source ~/.bashrc  # or source ~/.zshrc

# 2. Add a command
lz add "python train.py --config /configs/model.yaml --epochs 100 --debug True"

# 3. Run it
lz
```

## Adding Commands

Paste any command into `lz add` and it walks you through each flag interactively, asking whether to keep it static, make it a directory picker, a value list, optional, etc.

```
$ lz add "python train.py --config /configs/model.yaml --epochs 100 --debug True --verbose"

Building command from: python train.py --config /configs/model.yaml --epochs 100 --debug True --verbose

Base command: python train.py
Found 4 flag(s) to configure

[1/4] Flag: --config = /configs/model.yaml
How should this flag's value be set?
    Keep static (always use this value)
  > Directory picker (browse and select a path)
    Value list (choose from predefined options)

Base directory [/configs]:
Filter pattern (e.g., *.yaml, empty for all): *.yaml

[2/4] Flag: --epochs = 100
How should this flag's value be set?
  > Value list (choose from predefined options)

Enter values one per line. Empty line to finish.
Value: 10
Value: 50
Value: 100
Value: ...
Value:

[3/4] Flag: --debug = True
  > Make dynamic (choose True/False at runtime)

[4/4] Flag: --verbose
  > Make optional (choose to include or skip at runtime)

Command name: train
Tags (comma-separated, optional): ML,Training

Added 'train': python train.py {%--config:/configs:*.yaml%} --epochs {%[10,50,100,...]%} {%--debug:[True,False]%} {%?--verbose%}
```

The command is now available as a shell alias and through the picker.

## The Picker

Run `lz` to launch the interactive picker. It shows all your saved commands and lets you search, select, and run them.

```
$ lz

Select command:
  > train        [ML, Training]  python train.py {%--config:/configs:*.yaml%} ...
    deploy       [K8s]           kubectl apply -f deploy/ {%?--dry-run:[client,server]%}
    gs           [Git]           git status
```

Once you select a command, any dynamic bindings are resolved through pickers in sequence (e.g., pick a config file, pick epoch count, etc.), then the command runs.

**Keybindings:**

| Key | Action |
|---|---|
| `Up/Down` or `j/k` | Navigate |
| `Enter` | Select and run |
| `/` | Filter/search (matches name, command, tags) |
| `e` | Add extra arguments before running |
| `m` | Modify selected command |
| `x` | Delete selected command |
| `c` | Enter custom value (when `...` in binding) |
| `s` | Skip optional binding |
| `q` / `Esc` | Cancel |

Filter by tag directly from the command line:

```bash
lz list -t ML
```

## Config

Everything lives in `~/.config/laziest/commands.json`. Sync it however you already sync your dotfiles.

## Reference

For the full dynamic binding syntax (directory pickers, value lists, custom input, optional bindings, etc.), see [REFERENCE.md](REFERENCE.md).
