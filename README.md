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

### Add commands

```bash
# Add with tags
laziest add train_model "python train.py" -t ModelTraining,FlowCreation
laziest add gs "git status" -t Git

# Pipe from stdin (useful for adding from shell history)
echo "kubectl get pods" | laziest add kgp -t K8s
```

### List commands

```bash
laziest              # List all commands with tags
laziest list -t Git  # Filter by tag
```

Example output:

```
  train_model  [ModelTraining, FlowCreation]  python train.py
  gs           [Git]                          git status
  kgp          [K8s]                          kubectl get pods
```

### Run commands

```bash
laziest run gs              # Run by name
laziest run -t ModelTraining  # Interactive picker if multiple matches
```

### Manage tags

```bash
laziest tags  # List all tags with command counts
```

### Remove commands

```bash
laziest rm gs  # Remove by name
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

### Multiple Bindings

Commands can have multiple bindings - pickers appear in sequence:

```bash
laziest add train "python train.py --config {%/configs:*.yaml%} --gpu {%[True,False]%}" -t ML
```

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
...
```

## Examples

```bash
# DevOps workflow
laziest add deploy "kubectl apply -f deploy/" -t K8s,Deploy
laziest add logs "kubectl logs -f deployment/app" -t K8s,Debug

# ML workflow
laziest add train "python train.py --config config.yaml" -t ML,Training
laziest add eval "python eval.py --checkpoint latest" -t ML,Eval

# Git shortcuts
laziest add gs "git status" -t Git
laziest add gp "git push origin HEAD" -t Git

# Run all ML commands via picker
laziest run -t ML
```
