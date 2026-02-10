# lz Reference

Full reference for dynamic bindings, command management, and advanced usage.

## Dynamic Bindings

Commands can include dynamic placeholders (`{%...%}`) that prompt for selection at runtime. Use `lz add` to create these interactively, or write them manually with `lz add-raw`.

### Directory Binding

Bind a parameter to files in a directory:

```bash
# All files in directory
lz add-raw train "python train.py --config {%/path/to/configs%}" -t ML

# With extension filter
lz add-raw train "python train.py --config {%/path/to/configs:*.yaml%}" -t ML

# With flag inside binding (flag is removed if binding is skipped)
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

When running:
- A `[Custom]` option appears at the end of the picker
- Press `c` to enter a custom value
- Or select `[Custom]` and press Enter

### Optional Bindings

Mark bindings as optional with `?` prefix. Optional bindings can be skipped at runtime:

```bash
# Optional value binding with flag inside placeholder
lz add-raw train "python train.py {%?--debug:[True,False]%}" -t ML

# Optional directory binding
lz add-raw build "docker build {%?--platform:/platforms:*.txt%} ." -t Docker

# Optional boolean flag (include or skip)
lz add-raw train "python train.py {%?--verbose%}" -t ML
```

When running:
- A `[Skip]` option appears in the picker
- Press `s` to skip the binding
- Skipping removes the entire placeholder (including embedded flags)

### Multiple Bindings

Commands can have multiple bindings -- pickers appear in sequence:

```bash
lz add-raw train "python train.py --config {%/configs:*.yaml%} {%?--debug:[True,False]%}" -t ML
```

## Adding Commands

### Interactive Builder (Recommended)

```bash
lz add "python train.py --config /configs/model.yaml --epochs 100 --debug True"
```

Walks you through each flag, asking how it should behave at runtime.

### Manual Syntax

```bash
# Add with tags
lz add-raw train_model "python train.py" -t ML
lz add-raw gs "git status" -t Git

# Pipe from stdin
echo "kubectl get pods" | lz add-raw kgp -t K8s
```

## Running Commands

```bash
lz                                 # Interactive picker
lz list -t ML                     # Picker filtered by tag
lz run gs                          # Run by name
lz run train_model --extra --verbose  # Run with extra args
lz run -t ML                       # Picker if multiple matches
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

## Modifying Commands

Press `m` in the interactive picker to modify a command's name, command string, or tags. All fields are optional -- press Enter to keep the current value.

## Deleting Commands

Press `x` in the interactive picker to delete a command (with `y/n` confirmation), or use the CLI:

```bash
lz rm gs
```

## Tags

Tags help organize commands by project or category:

- Comma-separated, no spaces: `-t Tag1,Tag2`
- Filter commands: `lz list -t Tag`
- Run with picker: `lz run -t Tag`
- Tags are displayed in the picker: `name  [Tag1, Tag2]  command`
- List all tags: `lz tags`

## How It Works

1. Commands are stored in `~/.config/laziest/commands.json`
2. Shell aliases are written to `~/.config/laziest/aliases.sh`
3. `lz init` adds a one-time source line to `.bashrc`/`.zshrc`
4. After adding a command, it's immediately available as a shell alias (after sourcing)
