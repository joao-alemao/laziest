# laziest

Quick command aliases manager with tagging support.

## Installation

### From GitHub Releases

Requires `gh` CLI for private repo authentication:

```bash
gh release download --repo joao-alemao/laziest --pattern '*linux_amd64*'
tar xzf laziest_linux_amd64.tar.gz
sudo mv laziest /usr/local/bin/
```

For ARM64:

```bash
gh release download --repo joao-alemao/laziest --pattern '*linux_arm64*'
tar xzf laziest_linux_arm64.tar.gz
sudo mv laziest /usr/local/bin/
```

### From Source

```bash
git clone git@github.com:joao-alemao/laziest.git
cd laziest
go build -o laziest ./cmd/laziest
sudo mv laziest /usr/local/bin/
```

## Setup

Run once to add shell integration:

```bash
laziest init
source ~/.bashrc  # or source ~/.zshrc
```

This adds a source line to your shell rc file that loads aliases from `~/.config/laziest/aliases.sh`.

## Usage

### Interactive Picker (Default)

Simply run `laziest` or `laziest list` to launch an interactive picker:

```bash
laziest              # Interactive picker with all commands
laziest list -t Git  # Interactive picker filtered by tag
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

### Add commands

```bash
# Add with tags
laziest add train_model "python train.py" -t ModelTraining,FlowCreation
laziest add gs "git status" -t Git

# Pipe from stdin (useful for adding from shell history)
echo "kubectl get pods" | laziest add kgp -t K8s
```

### Run commands

```bash
laziest run gs                          # Run by name
laziest run train_model --extra --verbose  # Run with extra args
laziest run -t ModelTraining            # Interactive picker if multiple matches
```

### Modify commands

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

### Delete commands

Press `x` in the interactive picker to delete a command:

```
Select command:
  > old_command  []  echo "delete me"

[Press x]

Delete 'old_command'? (y/n)
```

Or use the CLI:

```bash
laziest rm gs  # Remove by name
```

### Manage tags

```bash
laziest tags  # List all tags with command counts
```

### Help

```bash
laziest help     # Show help
laziest version  # Show version
```

## How It Works

1. Commands are stored in `~/.config/laziest/commands.json`
2. Shell aliases are written to `~/.config/laziest/aliases.sh`
3. `laziest init` adds a one-time source line to `.bashrc`/`.zshrc`
4. After adding a command, it's immediately available as a shell alias (after sourcing)

## Tags

Tags help organize commands by project or category:

- Comma-separated, no spaces: `-t Tag1,Tag2`
- Filter commands: `laziest list -t Tag`
- Run with picker: `laziest run -t Tag` (shows interactive picker if multiple matches)
- Tags are displayed in the picker: `command_name  [Tag1, Tag2]  actual command`

## Dynamic Bindings

Commands can include dynamic placeholders that prompt for selection at runtime.

### Directory Binding

Bind a parameter to files in a directory:

```bash
# All files in directory
laziest add train "python train.py --config {%/path/to/configs%}" -t ML

# With extension filter
laziest add train "python train.py --config {%/path/to/configs:*.yaml%}" -t ML
```

When run, shows a picker with matching files (searched recursively). The selected file's absolute path is used.

### Value Binding

Bind a parameter to a fixed set of values:

```bash
laziest add deploy "kubectl apply --dry-run={%[none,client,server]%}" -t K8s
laziest add train "python train.py --use-gpu {%[True,False]%}" -t ML
```

### Custom Input Binding

Add `...` to a value binding to allow custom user input in addition to predefined values:

```bash
# Predefined values + custom input option
laziest add epochs "python train.py --epochs {%[10,50,100,...]%}" -t ML

# Custom input only (no predefined values)
laziest add msg "echo {%[...]%}" -t Util
```

When running commands with custom input bindings:
- A `[Custom]` option appears at the end of the picker
- Press `c` to enter a custom value
- Or select `[Custom]` and press Enter

### Optional Bindings

Mark bindings as optional with `?` prefix. Optional bindings can be skipped:

```bash
# Optional value binding with flag inside placeholder
laziest add train "python train.py {%?--debug:[True,False]%}" -t ML

# Optional directory binding
laziest add build "docker build {%?--platform:/platforms:*.txt%} ." -t Docker
```

When running commands with optional bindings:
- A `[Skip]` option appears in the picker
- Press `s` to skip the binding
- Skipping removes the entire placeholder (including embedded flags)

### Multiple Bindings

Commands can have multiple bindings - pickers appear in sequence:

```bash
laziest add train "python train.py --config {%/configs:*.yaml%} {%?--debug:[True,False]%}" -t ML
```

### Extra Arguments

Append additional arguments to any command at runtime:

```bash
# Via --extra flag
laziest run train --extra --verbose --epochs 100

# Via 'e' key in interactive picker
laziest       # Press 'e', then type extra args
```

Extra arguments are always appended to the end of the resolved command.

### Shell Aliases for Bound Commands

Commands with bindings create aliases that invoke `laziest run`, so you still get the interactive pickers:

```bash
# Generated alias:
alias train='laziest run train'

# Usage - just type the alias name:
$ train
Select file for --config [/path/to/configs]:
> model_v1.yaml
  model_v2.yaml

Select value for --debug:
  [Skip]
> True
  False

Extra arguments: --verbose
...
```

## Examples

```bash
# DevOps workflow with optional debug flag
laziest add deploy "kubectl apply -f deploy/ {%?--dry-run:[client,server]%}" -t K8s,Deploy
laziest add logs "kubectl logs -f deployment/app" -t K8s,Debug

# ML workflow with bindings and extra args
laziest add train "python train.py --config {%/configs:*.yaml%} {%?--debug:[True,False]%}" -t ML,Training
laziest add eval "python eval.py --checkpoint latest" -t ML,Eval
laziest run train --extra --epochs 100  # Add extra args

# Custom input for epochs with preset suggestions
laziest add epochs "python train.py --epochs {%[10,50,100,...]%}" -t ML

# Git shortcuts
laziest add gs "git status" -t Git
laziest add gp "git push origin HEAD" -t Git

# Interactive picker workflows
laziest           # Pick any command
laziest list -t ML  # Pick from ML commands only

# Use filter to find commands quickly
laziest           # Then press '/' and type 'git' to filter
```
