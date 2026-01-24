# lz

Quick command aliases manager with dynamic bindings. Convert any command into a reusable alias with interactive parameter selection.

## Why lz?

- **Zero learning curve** - Paste any command into `lz add` and interactively convert it to a dynamic alias. No special syntax to memorize.

- **Just works** - Built-in picker, automatic shell integration, no external dependencies like fzf or peco required.

- **Powerful when you need it** - Directory pickers with glob filters, value lists with custom input, optional flags that cleanly disappear when skipped.

- **Simple to backup** - Everything lives in `~/.config/laziest/commands.json`. Sync it however you already sync your dotfiles.

Unlike snippet managers that require learning a new syntax or cheatsheet tools focused on documentation, lz is designed for your personal command workflows - take a command you just ran, make parts of it dynamic, and never type it out again.

## Quick Start

```bash
# Install and setup
lz init
source ~/.bashrc  # or source ~/.zshrc

# Add a command interactively (recommended)
lz add "python train.py --config /configs/model.yaml --epochs 100 --debug True"

# Run it
lz
```

## Interactive Command Builder (Default)

The default `lz add` command walks you through each flag interactively:

```bash
lz add "python train.py --config /configs/model.yaml --epochs 100 --debug True --verbose"
```

```
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
Make this flag optional? (y/n): no

[2/4] Flag: --epochs = 100
How should this flag's value be set?
    Keep static (always use this value)
  > Value list (choose from predefined options)

Enter values one per line. Empty line to finish.
Tip: Add '...' as the last value to allow custom input at runtime.
Suggested: 100
Value: 10
Value: 50
Value: 100
Value: ...
Value: 
Make this flag optional? (y/n): no

[3/4] Flag: --debug = True
How should this flag behave?
    Keep static (always use True)
  > Make dynamic (choose True/False at runtime)
    Make optional + dynamic (choose True/False or skip entirely)

[4/4] Flag: --verbose
How should this flag behave?
    Keep static (always include this flag)
  > Make optional (choose to include or skip at runtime)

----------------------------------------
Generated command:
  python train.py {%--config:/configs:*.yaml%} --epochs {%[10,50,100,...]%} {%--debug:[True,False]%} {%?--verbose%}

Command name: train
Tags (comma-separated, optional): ML,Training

Added 'train': python train.py {%--config:/configs:*.yaml%} --epochs {%[10,50,100,...]%} {%--debug:[True,False]%} {%?--verbose%}
Tags: ML, Training
```

### Flag Types and Options

**Value flags** (e.g., `--epochs 100`):
- Keep static: Always use this value
- Directory picker: Browse and select a path at runtime
- Value list: Choose from predefined options

**True/False flags** (e.g., `--debug True`):
- Keep static: Always use this value
- Make dynamic: Choose True/False at runtime
- Make optional + dynamic: Choose True/False or skip entirely

**Boolean flags** (no value, e.g., `--verbose`):
- Keep static: Always include the flag
- Make optional: Choose to include or skip at runtime

## Installation

### From GitHub Releases

Requires `gh` CLI for private repo authentication:

```bash
gh release download --repo joao-alemao/laziest --pattern '*linux_amd64*'
tar xzf lz_linux_amd64.tar.gz
sudo mv lz /usr/local/bin/
```

For ARM64:

```bash
gh release download --repo joao-alemao/laziest --pattern '*linux_arm64*'
tar xzf lz_linux_arm64.tar.gz
sudo mv lz /usr/local/bin/
```

### From Source

```bash
git clone git@github.com:joao-alemao/laziest.git
cd laziest
go build -o lz ./cmd/laziest
sudo mv lz /usr/local/bin/
```

## Setup

Run once to add shell integration:

```bash
lz init
source ~/.bashrc  # or source ~/.zshrc
```

This adds a source line to your shell rc file that loads aliases from `~/.config/laziest/aliases.sh`.

## Usage

### Interactive Picker (Default)

Simply run `lz` or `lz list` to launch an interactive picker:

```bash
lz              # Interactive picker with all commands
lz list -t Git  # Interactive picker filtered by tag
```

**Picker keys:**
- `↑/↓` or `j/k` - Navigate
- `Enter` - Select and run command
- `/` - Filter commands (search by name, command, or tags)
- `e` - Add extra arguments, then run
- `m` - Modify selected command (rename, change command, change tags)
- `x` - Delete selected command (with confirmation)
- `c` - Enter custom value (when `...` in binding)
- `s` - Skip optional binding
- `q` or `Esc` - Cancel

**Filter mode (`/`):**
- Type to filter commands in real-time (case-insensitive)
- Matches against command name, command text, and tags
- `Esc` - Clear filter and return to full list
- `Enter` - Select current item
- `Ctrl+C` - Cancel picker

### Add Commands (Manual Syntax)

For simple commands or when you want full control over binding syntax:

```bash
# Add with tags
lz add-raw train_model "python train.py" -t ML
lz add-raw gs "git status" -t Git

# Pipe from stdin (useful for adding from shell history)
echo "kubectl get pods" | lz add-raw kgp -t K8s
```

### Run Commands

```bash
lz run gs                          # Run by name
lz run train_model --extra --verbose  # Run with extra args
lz run -t ML                       # Interactive picker if multiple matches
```

### Modify Commands

Press `m` in the interactive picker to modify a command:

```
Select command:
  > train_model  [ML]  python train.py
    gs           [Git] git status

[Press m]

New name [train_model]: train_v2
New command [python train.py]: python train_v2.py --config config.yaml
New tags [ML]: ML,Training

Modified 'train_model' -> 'train_v2'
```

All fields are optional - press Enter to keep the current value.

### Delete Commands

Press `x` in the interactive picker to delete a command:

```
Select command:
  > old_command  []  echo "delete me"

[Press x]

Delete 'old_command'? (y/n)
```

Or use the CLI:

```bash
lz rm gs  # Remove by name
```

### Manage Tags

```bash
lz tags  # List all tags with command counts
```

### Help

```bash
lz help     # Show help
lz version  # Show version
```

## How It Works

1. Commands are stored in `~/.config/laziest/commands.json`
2. Shell aliases are written to `~/.config/laziest/aliases.sh`
3. `lz init` adds a one-time source line to `.bashrc`/`.zshrc`
4. After adding a command, it's immediately available as a shell alias (after sourcing)

## Dynamic Bindings Reference

Commands can include dynamic placeholders that prompt for selection at runtime. Use `lz add` to create these interactively, or write them manually with `lz add-raw`:

### Directory Binding

Bind a parameter to files in a directory:

```bash
# All files in directory
lz add-raw train "python train.py --config {%/path/to/configs%}" -t ML

# With extension filter
lz add-raw train "python train.py --config {%/path/to/configs:*.yaml%}" -t ML

# With flag inside binding
lz add-raw train "python train.py {%--config:/path/to/configs:*.yaml%}" -t ML
```

When run, shows a picker with matching files (searched recursively). The selected file's absolute path is used.

### Value Binding

Bind a parameter to a fixed set of values:

```bash
lz add-raw deploy "kubectl apply --dry-run={%[none,client,server]%}" -t K8s
lz add-raw train "python train.py {%--use-gpu:[True,False]%}" -t ML
```

### Custom Input Binding

Add `...` to a value binding to allow custom user input in addition to predefined values:

```bash
# Predefined values + custom input option
lz add-raw epochs "python train.py --epochs {%[10,50,100,...]%}" -t ML

# Custom input only (no predefined values)
lz add-raw msg "echo {%[...]%}" -t Util
```

When running commands with custom input bindings:
- A `[Custom]` option appears at the end of the picker
- Press `c` to enter a custom value
- Or select `[Custom]` and press Enter

### Optional Bindings

Mark bindings as optional with `?` prefix. Optional bindings can be skipped:

```bash
# Optional value binding with flag inside placeholder
lz add-raw train "python train.py {%?--debug:[True,False]%}" -t ML

# Optional directory binding
lz add-raw build "docker build {%?--platform:/platforms:*.txt%} ." -t Docker

# Optional boolean flag (include or skip)
lz add-raw train "python train.py {%?--verbose%}" -t ML
```

When running commands with optional bindings:
- A `[Skip]` option appears in the picker
- Press `s` to skip the binding
- Skipping removes the entire placeholder (including embedded flags)

### Multiple Bindings

Commands can have multiple bindings - pickers appear in sequence:

```bash
lz add-raw train "python train.py --config {%/configs:*.yaml%} {%?--debug:[True,False]%}" -t ML
```

### Extra Arguments

Append additional arguments to any command at runtime:

```bash
# Via --extra flag
lz run train --extra --verbose --epochs 100

# Via 'e' key in interactive picker
lz       # Press 'e', then type extra args
```

Extra arguments are always appended to the end of the resolved command.

## Tags

Tags help organize commands by project or category:

- Comma-separated, no spaces: `-t Tag1,Tag2`
- Filter commands: `lz list -t Tag`
- Run with picker: `lz run -t Tag` (shows interactive picker if multiple matches)
- Tags are displayed in the picker: `command_name  [Tag1, Tag2]  actual command`

## Examples

```bash
# Add commands interactively (recommended)
lz add "python train.py --config /configs/model.yaml --epochs 100 --debug True"
lz add "kubectl apply -f deploy/ --dry-run server --namespace prod"

# Or add manually with binding syntax
lz add-raw deploy "kubectl apply -f deploy/ {%?--dry-run:[client,server]%}" -t K8s
lz add-raw train "python train.py --config {%/configs:*.yaml%} {%?--debug:[True,False]%}" -t ML

# Simple aliases
lz add-raw gs "git status" -t Git
lz add-raw gp "git push origin HEAD" -t Git

# Run commands
lz                     # Interactive picker
lz list -t ML          # Filter by tag
lz run train --extra --verbose  # With extra args

# Use filter to find commands quickly
lz                     # Then press '/' and type to filter
```
