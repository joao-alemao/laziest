# laziest (lz) - Quick command aliases with dynamic bindings
# Demo tape for VHS (https://github.com/charmbracelet/vhs)

Output lzdemo.gif

Require lz

Set Shell "bash"
Set FontSize 26
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set WindowBarSize 40
Set TypingSpeed 50ms
Set Padding 5

# ============================================================
# Hidden setup: clean config, create demo directory structure
# ============================================================
Hide
Type "rm -rf /tmp/lz-demo && mkdir -p /tmp/lz-demo/configs"
Enter
Sleep 300ms
Type `echo "lr: 0.001" > /tmp/lz-demo/configs/base.yaml`
Enter
Sleep 200ms
Type `echo "lr: 0.0001" > /tmp/lz-demo/configs/large_model.yaml`
Enter
Sleep 200ms
Type `echo "lr: 0.01" > /tmp/lz-demo/configs/small_model.yaml`
Enter
Sleep 200ms
# Create a no-op train.py so the command runs cleanly
Type `printf 'import argparse\np = argparse.ArgumentParser()\np.add_argument("--config")\np.add_argument("--epochs", type=int)\np.add_argument("--debug")\np.add_argument("--verbose", action="store_true")\na = p.parse_args()\nprint(f"Training with config={a.config} epochs={a.epochs} debug={a.debug} verbose={a.verbose}")\n' > /tmp/lz-demo/train.py`
Enter
Sleep 200ms
# Backup existing config
Type "cp ~/.config/laziest/commands.json ~/.config/laziest/commands.json.bak 2>/dev/null; true"
Enter
Sleep 200ms
Type "cp ~/.config/laziest/aliases.sh ~/.config/laziest/aliases.sh.bak 2>/dev/null; true"
Enter
Sleep 200ms
# Start fresh
Type `mkdir -p ~/.config/laziest && echo '{"commands":[]}' > ~/.config/laziest/commands.json`
Enter
Sleep 200ms
Type "clear"
Enter
Sleep 500ms
Show

# ============================================================
# Scene 0: Life WITHOUT laziest - the pain
# ============================================================
Sleep 500ms

# User types a long training command from memory, struggling
Type "python /tmp/lz-demo/train.py --config /tmp/lz-demo/configs/base.yaml --epochs 100 --debug True --verbose"
Sleep 1.5s
Enter
Sleep 2s

# Some time passes... user needs to run it again
# Try to recall the command, get it wrong
Type "python /tmp/lz-demo/train.py --config"
Sleep 1s
# Hmm, what was the path?
Type " /tmp/lz-demo/configs/"
Sleep 800ms
# Which config was it?
Type "base"
Sleep 500ms
Backspace 4
Type "large_model.yaml"
Sleep 600ms
# What were the other flags again?
Type " --epochs"
Sleep 800ms
# Was it 50 or 100?
Type " 50 --debug"
Sleep 600ms
# Wait... True or False? And was there another flag?
Ctrl+C
Sleep 1s

# Try scrolling through history
Ctrl+R
Sleep 800ms
Type "train"
Sleep 1.5s
Ctrl+C
Sleep 1s

# Frustrated, clear the screen
Type "clear"
Enter
Sleep 1s

# The segue
Type "# ... what was that command again?"
Sleep 2s
Enter
Type "# and which options did it support?"
Sleep 2s
Enter
Sleep 1s
Type "# Enter: laziest (lz)"
Sleep 2s
Enter
Sleep 1.5s
Type "clear"
Enter
Sleep 1s

# ============================================================
# Scene 1: lz add - Interactive command builder (flagship)
# ============================================================
Sleep 500ms
Type `lz add "python /tmp/lz-demo/train.py --config /tmp/lz-demo/configs/base.yaml --epochs 100 --debug True --verbose"`
Sleep 1.5s
Enter
Sleep 1.5s

# Flag 1/4: --config = /tmp/lz-demo/configs/base.yaml
# Picker shows: Keep static | Directory picker | Value list
# Select "Directory picker" (Down once, Enter)
Down
Sleep 600ms
Enter
Sleep 1s

# "Base directory: /tmp/lz-demo/configs" (pre-filled, accept default)
Enter
Sleep 1s

# "Filter pattern: " -> type *.yaml
Type "*.yaml"
Sleep 600ms
Enter
Sleep 1s

# "Make this flag optional? (y/n):" -> n
Type "n"
Sleep 1s

# Flag 2/4: --epochs = 100
# Picker shows: Keep static | Directory picker | Value list
# Select "Value list" (Down twice, Enter)
Sleep 1s
Down
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 1s

# Enter values one per line
Type "10"
Enter
Sleep 600ms
Type "50"
Enter
Sleep 600ms
Type "100"
Enter
Sleep 600ms
Type "..."
Enter
Sleep 600ms
# Empty line to finish
Enter
Sleep 1s

# "Make this flag optional? (y/n):" -> n
Type "n"
Sleep 1s

# Flag 3/4: --debug = True
# Picker shows: Keep static | Make dynamic | Make optional + dynamic
# Select "Make dynamic" (Down once, Enter)
Sleep 1s
Down
Sleep 500ms
Enter
Sleep 1s

# Flag 4/4: --verbose (boolean, no value)
# Picker shows: Keep static | Make optional
# Select "Make optional" (Down once, Enter)
Sleep 1s
Down
Sleep 500ms
Enter
Sleep 1.5s

# Shows generated command, now enter name and tags
# "Command name: " -> type train
Sleep 1s
Type "train"
Sleep 600ms
Enter
Sleep 1s

# "Tags (comma-separated, optional): " -> type ML
Type "ML"
Sleep 600ms
Enter
Sleep 2s

# ============================================================
# Scene 2: lz add-raw - Quick alias
# ============================================================
Type `lz add-raw gs "git status" -t Git`
Sleep 500ms
Enter
Sleep 1.5s

# ============================================================
# Scene 3: lz tags - Show organization
# ============================================================
Type "lz tags"
Sleep 500ms
Enter
Sleep 2s

# ============================================================
# Scene 4: lz (interactive picker) - select and run with bindings
# ============================================================
Type "lz"
Sleep 500ms
Enter
Sleep 1s

# Picker shows: train, gs
# Navigate down to gs, then back up to train
Down
Sleep 500ms
Up
Sleep 500ms
# Select train
Enter
Sleep 800ms

# Binding 1: Select config file
# Shows yaml files: base.yaml, large_model.yaml, small_model.yaml
# Pick large_model.yaml (Down once, Enter)
Down
Sleep 300ms
Enter
Sleep 800ms

# Binding 2: Select epochs
# Shows: 10, 50, 100, [Custom]
# Pick 50 (Down once, Enter)
Down
Sleep 300ms
Enter
Sleep 800ms

# Binding 3: Select debug
# Shows: True, False
# Pick False (Down once, Enter)
Down
Sleep 300ms
Enter
Sleep 800ms

# Binding 4: Include --verbose? (y/n)
# Type n to skip
Type "n"
Sleep 2s

# ============================================================
# Scene 5: lz picker with / filter
# ============================================================
Type "lz"
Sleep 500ms
Enter
Sleep 1s

# Enter filter mode
Type "/"
Sleep 500ms
# Type search query
Type "git"
Sleep 1s
# Enter to select filtered result
Enter
Sleep 2.5s

# ============================================================
# Scene 6: Shell aliases - commands available without invoking lz
# ============================================================
Type "lz init"
Sleep 500ms
Enter
Sleep 1.5s

Type "source ~/.bashrc"
Sleep 500ms
Enter
Sleep 1.5s

# Now "gs" works directly as an alias - no lz needed
Type "gs"
Sleep 1s
Enter
Sleep 2s

# "train" also works directly - triggers interactive bindings
Type "train"
Sleep 1s
Enter
Sleep 1s

# Binding 1: Select config file
Down
Sleep 300ms
Enter
Sleep 800ms

# Binding 2: Select epochs
Down
Sleep 300ms
Enter
Sleep 800ms

# Binding 3: Select debug
Down
Sleep 300ms
Enter
Sleep 800ms

# Binding 4: Include --verbose? (y/n)
Type "n"
Sleep 2.5s

# ============================================================
# Hidden cleanup: restore original config, remove temp files
# ============================================================
Hide
Type "rm -rf /tmp/lz-demo"
Enter
Sleep 200ms
Type "cp ~/.config/laziest/commands.json.bak ~/.config/laziest/commands.json 2>/dev/null; true"
Enter
Sleep 200ms
Type "cp ~/.config/laziest/aliases.sh.bak ~/.config/laziest/aliases.sh 2>/dev/null; true"
Enter
Sleep 200ms
Type "source ~/.bashrc"
Enter
Sleep 200ms
