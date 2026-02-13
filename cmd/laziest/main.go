package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"laziest/internal/binding"
	"laziest/internal/builder"
	"laziest/internal/config"
	"laziest/internal/picker"
	"laziest/internal/shell"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		cmdInteractiveList(nil)
		os.Exit(0)
	}

	cmd := os.Args[1]

	switch cmd {
	case "list", "ls", "l":
		tags, _ := parseTagsFlag(os.Args[2:])
		cmdInteractiveList(tags)
	case "add", "a":
		cmdAdd(os.Args[2:])
	case "add-raw", "ar":
		cmdAddRaw(os.Args[2:])
	case "run", "r":
		cmdRun(os.Args[2:])
	case "last":
		cmdLast()
	case "remove", "rm":
		cmdRemove(os.Args[2:])
	case "tags", "t":
		cmdTags()
	case "init":
		cmdInit()
	case "help", "-h", "--help":
		printUsage()
	case "version", "-v", "--version":
		fmt.Printf("lz version %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`lz - Quick command aliases manager

Usage:
  lz                           Interactive command picker
  lz list [-t <tag>]           Interactive picker, optionally filter by tag
  lz add "<cmd>"               Interactive command builder from example
  lz add-raw <name> <cmd> [-t <tags>]  Add command with manual binding syntax
  lz run <name> [--extra <args>]   Run command by name
  lz run -t <tag> [--extra <args>] Pick and run a command with that tag
  lz last                      Pick and run from recent commands
  lz remove <name>             Remove a command
  lz tags                      List all tags with command counts
  lz init                      One-time setup: add source line to shell rc
  lz help                      Show this help
  lz version                   Show version

Adding commands (interactive builder - recommended):
  lz add "python train.py --config /configs/model.yaml --epochs 100"
  
  Walks through each flag and asks how to handle it:
  - Keep static: Flag value stays as-is
  - Directory picker: Browse and select a path at runtime
  - Value list: Choose from predefined options at runtime
  - Optional boolean: Include or skip the flag at runtime

Adding commands (manual syntax):
  lz add-raw deploy "kubectl apply -f ." -t DevOps,K8s
  echo "git status" | lz add-raw gs -t Git

Tags:
  - Comma-separated, no spaces: -t Tag1,Tag2
  - Used for filtering and organizing commands

Dynamic bindings (for add-raw):
  Directory binding:  {%/path/to/dir%} or {%/path/to/dir:*.yaml%}
  Value binding:      {%[val1,val2,val3]%}
  Custom input:       {%[val1,val2,...]%} - allows custom value via [Custom] option
  Optional binding:   {%?...%} or {%?--flag:...%}
  
  Commands with bindings prompt for selection at runtime.
  Optional bindings show [Skip] option. Press 's' to skip.
  Custom input bindings show [Custom] option. Press 'c' for custom value.
  Skipping removes both the flag and placeholder from the command.

Extra arguments:
  Use --extra flag or press 'e' in picker to append extra args to command.
  Example: lz run train --extra --verbose --epochs 100

Interactive picker keys:
  ↑/↓ or j/k   Navigate
  Enter        Select and run
  e            Add extra args then run
  c            Enter custom value (when ... in binding)
  s            Skip optional binding
  q or Esc     Cancel

Examples:
  lz add "python train.py --config /configs/model.yaml --epochs 100"
  lz add-raw gs "git status" -t Git
  lz add-raw train "python train.py --config {%/configs:*.yaml%}" -t ML
  lz add-raw deploy "kubectl apply --dry-run={%[none,client,server]%}" -t K8s
  lz run gs
  lz run train --extra --verbose
  lz run -t ML
  lz list -t Git
  lz rm gs`)
}

func cmdInit() {
	updated, err := shell.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	if len(updated) == 0 {
		fmt.Println("lz is already configured in your shell rc files.")
		fmt.Println("If aliases aren't working, try: source ~/.bashrc or source ~/.zshrc")
		return
	}

	fmt.Println("Added source line to:")
	for _, path := range updated {
		fmt.Printf("  - %s\n", path)
	}
	fmt.Println()
	fmt.Println("Run 'source ~/.bashrc' or 'source ~/.zshrc' to activate.")
}

func cmdLast() {
	// Load history
	entries, err := config.LoadHistory()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading history: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("No recent commands.")
		fmt.Println("Run commands with 'lz' or 'lz run <name>' first.")
		return
	}

	// Build picker items with formatted display
	// Format: "command                    2m ago"
	maxCmdLen := 50

	items := make([]picker.Item, len(entries))
	for i, e := range entries {
		cmd := e.Command
		if len(cmd) > maxCmdLen {
			cmd = cmd[:maxCmdLen-3] + "..."
		}
		timeStr := formatRelativeTime(e.Timestamp)
		// Format with padding for alignment
		display := fmt.Sprintf("%-*s  %s", maxCmdLen, cmd, timeStr)
		items[i] = picker.Item{
			Name:    display,
			Command: e.Command, // Store the actual command here
			Tags:    []string{},
		}
	}

	// Show picker
	result := picker.Pick(items, "Recent commands:")

	if result.Action == picker.ActionCancel {
		return
	}

	// The selected item's Command field has the actual command to run
	selectedCmd := result.Value
	// Find the actual command from entries by matching the display name
	var actualItem picker.Item
	for _, item := range items {
		if item.Name == selectedCmd {
			actualItem = item
			break
		}
	}

	if actualItem.Command == "" {
		fmt.Fprintln(os.Stderr, "Error: could not find selected command")
		os.Exit(1)
	}

	// Execute the command
	fmt.Printf("Running: %s\n", actualItem.Command)
	fmt.Println(strings.Repeat("-", 40))

	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "/bin/sh"
	}

	execCmd := exec.Command(shellPath, "-c", actualItem.Command)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}

	// Update execution time.
	config.AddHistoryEntry(actualItem.Command, actualItem.Name)
}

func formatRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func cmdTags() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	counts := cfg.GetTagCounts()
	if len(counts) == 0 {
		fmt.Println("No tags defined. Add tags with: lz add-raw <name> <cmd> -t <tags>")
		return
	}

	// Sort tags alphabetically
	tags := make([]string, 0, len(counts))
	for tag := range counts {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	fmt.Println("Tags:")
	fmt.Println()
	for _, tag := range tags {
		fmt.Printf("  %-20s (%d commands)\n", tag, counts[tag])
	}
}

func cmdList(filterTags []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Commands) == 0 {
		fmt.Println("No commands saved.")
		fmt.Println()
		fmt.Println("Get started:")
		fmt.Println("  1. Run 'lz init' to set up shell integration")
		fmt.Println("  2. Add commands with 'lz add \"<command>\"'")
		return
	}

	// Filter commands if tag specified
	var commands []config.Command
	if len(filterTags) > 0 {
		// Get commands matching any of the filter tags
		seen := make(map[string]bool)
		for _, tag := range filterTags {
			for _, cmd := range cfg.GetCommandsByTag(tag) {
				if !seen[cmd.Name] {
					seen[cmd.Name] = true
					commands = append(commands, cmd)
				}
			}
		}
		if len(commands) == 0 {
			fmt.Printf("No commands found with tag(s): %s\n", strings.Join(filterTags, ", "))
			return
		}
	} else {
		commands = cfg.Commands
	}

	// Find max lengths for formatting
	maxNameLen := 0
	maxTagLen := 0
	for _, cmd := range commands {
		if len(cmd.Name) > maxNameLen {
			maxNameLen = len(cmd.Name)
		}
		tagStr := formatTags(cmd.Tags)
		if len(tagStr) > maxTagLen {
			maxTagLen = len(tagStr)
		}
	}

	// Print commands
	fmt.Println()
	for _, cmd := range commands {
		tagStr := formatTags(cmd.Tags)
		if tagStr != "" {
			fmt.Printf("  %-*s  %-*s  %s\n", maxNameLen, cmd.Name, maxTagLen, tagStr, cmd.Command)
		} else {
			fmt.Printf("  %-*s  %-*s  %s\n", maxNameLen, cmd.Name, maxTagLen, "", cmd.Command)
		}
	}
	fmt.Println()
}

func cmdInteractiveList(filterTags []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Commands) == 0 {
		fmt.Println("No commands saved.")
		fmt.Println()
		fmt.Println("Get started:")
		fmt.Println("  1. Run 'lz init' to set up shell integration")
		fmt.Println("  2. Add commands with 'lz add \"<command>\"'")
		return
	}

	for {
		// Filter commands if tag specified
		var commands []config.Command
		if len(filterTags) > 0 {
			// Get commands matching any of the filter tags
			seen := make(map[string]bool)
			for _, tag := range filterTags {
				for _, cmd := range cfg.GetCommandsByTag(tag) {
					if !seen[cmd.Name] {
						seen[cmd.Name] = true
						commands = append(commands, cmd)
					}
				}
			}
			if len(commands) == 0 {
				fmt.Printf("No commands found with tag(s): %s\n", strings.Join(filterTags, ", "))
				return
			}
		} else {
			commands = cfg.Commands
		}

		if len(commands) == 0 {
			fmt.Println("No commands left.")
			return
		}

		// Build picker items
		items := make([]picker.Item, len(commands))
		for i, cmd := range commands {
			items[i] = picker.Item{Name: cmd.Name, Command: cmd.Command, Tags: cmd.Tags}
		}

		// Show picker
		var promptStr string
		if len(filterTags) > 0 {
			promptStr = fmt.Sprintf("Select command [%s]:", strings.Join(filterTags, ", "))
		} else {
			promptStr = "Select command:"
		}

		result := picker.Pick(items, promptStr)

		// Handle delete action
		if result.Action == picker.ActionDelete {
			if err := cfg.RemoveCommandByName(result.Value); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := cfg.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				os.Exit(1)
			}
			if err := shell.UpdateAliases(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
			fmt.Printf("Deleted '%s'\n", result.Value)
			// Loop back to picker
			continue
		}

		// Handle modify action
		if result.Action == picker.ActionModify {
			// Validate new name if changed
			if result.NewName != result.Value {
				if !isValidAliasName(result.NewName) {
					fmt.Fprintf(os.Stderr, "Error: invalid alias name '%s'\n", result.NewName)
					fmt.Fprintln(os.Stderr, "Name must start with a letter and contain only letters, numbers, and underscores")
					continue
				}
				// Check for name conflict
				if _, err := cfg.GetCommandByName(result.NewName); err == nil {
					fmt.Fprintf(os.Stderr, "Error: command '%s' already exists\n", result.NewName)
					continue
				}
			}

			// Parse new tags
			var newTags []string
			if result.NewTags != "" {
				for _, t := range strings.Split(result.NewTags, ",") {
					t = strings.TrimSpace(t)
					if t != "" {
						newTags = append(newTags, t)
					}
				}
			}

			// Update the command
			if err := cfg.UpdateCommand(result.Value, result.NewName, result.NewCommand, newTags); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			if err := cfg.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				os.Exit(1)
			}
			if err := shell.UpdateAliases(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
			if result.NewName != result.Value {
				fmt.Printf("Modified '%s' -> '%s'\n", result.Value, result.NewName)
			} else {
				fmt.Printf("Modified '%s'\n", result.NewName)
			}
			// Loop back to picker
			continue
		}

		if result.Action == picker.ActionCancel {
			return
		}

		// Get the selected command
		cmd, err := cfg.GetCommandByName(result.Value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Resolve bindings and run
		finalCommand := cmd.Command
		extraArgs := ""

		// Handle extra args from picker
		if result.Action == picker.ActionSelectWithExtra {
			extraArgs = result.Extra
		}

		// Parse and resolve any bindings
		bindings, err := binding.Parse(cmd.Command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing bindings: %v\n", err)
			os.Exit(1)
		}

		for _, b := range bindings {
			var selected string
			prompt := binding.ExtractPromptContext(finalCommand, b)

			if b.Type == binding.BindingDirectory {
				// List files and show picker
				files, err := binding.ListFiles(b)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}

				bindResult := picker.PickString(files, prompt, b.Optional, false)
				if bindResult.Action == picker.ActionCancel {
					os.Exit(0) // User cancelled
				}
				if bindResult.Action == picker.ActionSkip {
					// Remove binding and flag from command
					finalCommand = binding.RemoveWithFlag(finalCommand, b)
					continue
				}
				// Use absolute path
				selected = binding.GetAbsolutePath(b, bindResult.Value)

			} else if b.Type == binding.BindingBooleanFlag {
				// Handle optional boolean flag - ask yes/no to include
				include, ok := picker.PromptYesNo(prompt)
				if !ok {
					os.Exit(0) // User cancelled
				}
				if !include {
					// User chose not to include - remove the flag
					finalCommand = binding.RemoveWithFlag(finalCommand, b)
					continue
				}
				// User chose to include - resolve with empty value (just the flag)
				selected = ""

			} else { // BindingValues
				bindResult := picker.PickString(b.Values, prompt, b.Optional, b.AllowCustom)
				if bindResult.Action == picker.ActionCancel {
					os.Exit(0) // User cancelled
				}
				if bindResult.Action == picker.ActionSkip {
					// Remove binding and flag from command
					finalCommand = binding.RemoveWithFlag(finalCommand, b)
					continue
				}
				selected = bindResult.Value
			}

			finalCommand = binding.Resolve(finalCommand, b, selected)
		}

		// Append extra args if provided
		if extraArgs != "" {
			finalCommand = finalCommand + " " + extraArgs
		}

		// Save to history for 'lz !!'
		config.AddHistoryEntry(finalCommand, cmd.Name)

		fmt.Printf("Running: %s\n", finalCommand)
		fmt.Println(strings.Repeat("-", 40))

		// Determine which shell to use
		shellPath := os.Getenv("SHELL")
		if shellPath == "" {
			shellPath = "/bin/sh"
		}

		execCmd := exec.Command(shellPath, "-c", finalCommand)
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		if err := execCmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
			os.Exit(1)
		}
		return
	}
}

func cmdAddRaw(args []string) {
	// Parse tags flag
	tags, remaining := parseTagsFlag(args)

	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Error: name required")
		fmt.Fprintln(os.Stderr, "Usage: lz add-raw <name> <command> [-t <tags>]")
		fmt.Fprintln(os.Stderr, "   or: echo 'command' | lz add-raw <name> [-t <tags>]")
		os.Exit(1)
	}

	name := remaining[0]

	// Validate name (must be valid for shell alias)
	if !isValidAliasName(name) {
		fmt.Fprintf(os.Stderr, "Error: invalid alias name '%s'\n", name)
		fmt.Fprintln(os.Stderr, "Name must start with a letter and contain only letters, numbers, and underscores")
		os.Exit(1)
	}

	var command string

	if len(remaining) >= 2 {
		// Command provided as argument
		command = strings.Join(remaining[1:], " ")
	} else {
		// Try to read from stdin
		var err error
		command, err = shell.ReadFromStdin()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: no command provided")
			fmt.Fprintln(os.Stderr, "Usage: lz add-raw <name> <command> [-t <tags>]")
			fmt.Fprintln(os.Stderr, "   or: echo 'command' | lz add-raw <name> [-t <tags>]")
			os.Exit(1)
		}
	}

	command = strings.TrimSpace(command)
	if command == "" {
		fmt.Fprintln(os.Stderr, "Error: command cannot be empty")
		os.Exit(1)
	}

	// Validate bindings in command
	bindings, err := binding.Parse(command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Warn about any issues with bindings
	for _, b := range bindings {
		for _, warning := range binding.Validate(b) {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.AddCommand(name, command, tags); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	// Update alias file
	if err := shell.UpdateAliases(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	fmt.Printf("Added '%s': %s\n", name, command)
	if len(tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(tags, ", "))
	}
}

func cmdAdd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: example command required")
		fmt.Fprintln(os.Stderr, "Usage: lz add \"<example command>\"")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  lz add \"python train.py --config /configs/model.yaml --epochs 100\"")
		os.Exit(1)
	}

	// Join args as the example command (handles both quoted and unquoted input)
	exampleCmd := strings.Join(args, " ")

	// Run interactive builder
	result := builder.BuildCommand(exampleCmd)
	if result.Cancelled {
		fmt.Println("Cancelled.")
		return
	}

	// Display the generated command
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("\033[1mGenerated command:\033[0m\n  %s\n\n", result.Command)

	// Validate bindings in command
	bindings, err := binding.Parse(result.Command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Warn about any issues with bindings
	for _, b := range bindings {
		for _, warning := range binding.Validate(b) {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
		}
	}

	// Prompt for name
	name, cancelled := picker.PromptInput("Command name: ", "")
	if cancelled || name == "" {
		fmt.Println("Cancelled.")
		return
	}

	// Validate name
	if !isValidAliasName(name) {
		fmt.Fprintf(os.Stderr, "Error: invalid alias name '%s'\n", name)
		fmt.Fprintln(os.Stderr, "Name must start with a letter and contain only letters, numbers, and underscores")
		os.Exit(1)
	}

	// Prompt for tags
	tagsInput, _ := picker.PromptInput("Tags (comma-separated, optional): ", "")
	var tags []string
	if tagsInput != "" {
		for _, t := range strings.Split(tagsInput, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	// Load config and add command
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.AddCommand(name, result.Command, tags); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	// Update alias file
	if err := shell.UpdateAliases(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	fmt.Printf("\nAdded '%s': %s\n", name, result.Command)
	if len(tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(tags, ", "))
	}
}

// parseExtraArgs splits args at --extra, returns (before, extraArgs)
func parseExtraArgs(args []string) ([]string, string) {
	for i, arg := range args {
		if arg == "--extra" {
			if i+1 < len(args) {
				return args[:i], strings.Join(args[i+1:], " ")
			}
			return args[:i], ""
		}
	}
	return args, ""
}

func cmdRun(args []string) {
	// Parse extra args first
	args, extraArgs := parseExtraArgs(args)

	// Parse tags flag
	tags, remaining := parseTagsFlag(args)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Commands) == 0 {
		fmt.Fprintln(os.Stderr, "No commands saved. Use 'lz add \"<command>\"' to add one.")
		os.Exit(1)
	}

	var cmd *config.Command

	// If tag specified, filter and possibly show picker
	if len(tags) > 0 {
		for {
			// Get commands matching the tag
			var matches []config.Command
			seen := make(map[string]bool)
			for _, tag := range tags {
				for _, c := range cfg.GetCommandsByTag(tag) {
					if !seen[c.Name] {
						seen[c.Name] = true
						matches = append(matches, c)
					}
				}
			}

			if len(matches) == 0 {
				fmt.Fprintf(os.Stderr, "No commands found with tag(s): %s\n", strings.Join(tags, ", "))
				os.Exit(1)
			}

			if len(matches) == 1 {
				cmd = &matches[0]
				break
			}

			// Show picker
			items := make([]picker.Item, len(matches))
			for i, m := range matches {
				items[i] = picker.Item{Name: m.Name, Command: m.Command, Tags: m.Tags}
			}

			result := picker.Pick(items, fmt.Sprintf("Select command [%s]:", strings.Join(tags, ", ")))

			// Handle delete action
			if result.Action == picker.ActionDelete {
				if err := cfg.RemoveCommandByName(result.Value); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				if err := cfg.Save(); err != nil {
					fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
					os.Exit(1)
				}
				if err := shell.UpdateAliases(cfg); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				}
				fmt.Printf("Deleted '%s'\n", result.Value)
				// Loop back to picker
				continue
			}

			// Handle modify action
			if result.Action == picker.ActionModify {
				// Validate new name if changed
				if result.NewName != result.Value {
					if !isValidAliasName(result.NewName) {
						fmt.Fprintf(os.Stderr, "Error: invalid alias name '%s'\n", result.NewName)
						fmt.Fprintln(os.Stderr, "Name must start with a letter and contain only letters, numbers, and underscores")
						continue
					}
					// Check for name conflict
					if _, err := cfg.GetCommandByName(result.NewName); err == nil {
						fmt.Fprintf(os.Stderr, "Error: command '%s' already exists\n", result.NewName)
						continue
					}
				}

				// Parse new tags
				var newTags []string
				if result.NewTags != "" {
					for _, t := range strings.Split(result.NewTags, ",") {
						t = strings.TrimSpace(t)
						if t != "" {
							newTags = append(newTags, t)
						}
					}
				}

				// Update the command
				if err := cfg.UpdateCommand(result.Value, result.NewName, result.NewCommand, newTags); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					continue
				}
				if err := cfg.Save(); err != nil {
					fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
					os.Exit(1)
				}
				if err := shell.UpdateAliases(cfg); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				}
				if result.NewName != result.Value {
					fmt.Printf("Modified '%s' -> '%s'\n", result.Value, result.NewName)
				} else {
					fmt.Printf("Modified '%s'\n", result.NewName)
				}
				// Loop back to picker
				continue
			}

			if result.Action == picker.ActionCancel {
				os.Exit(0) // User cancelled
			}

			// Handle extra args from picker
			if result.Action == picker.ActionSelectWithExtra {
				if extraArgs != "" {
					extraArgs = extraArgs + " " + result.Extra
				} else {
					extraArgs = result.Extra
				}
			}

			// Find the selected command
			cmd, _ = cfg.GetCommandByName(result.Value)
			break
		}
	} else if len(remaining) > 0 {
		// Run by name
		cmd, err = cfg.GetCommandByName(remaining[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Error: name or -t <tag> required")
		fmt.Fprintln(os.Stderr, "Usage: lz run <name>")
		fmt.Fprintln(os.Stderr, "   or: lz run -t <tag>")
		os.Exit(1)
	}

	// Execute the command
	finalCommand := cmd.Command

	// Parse and resolve any bindings
	bindings, err := binding.Parse(cmd.Command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing bindings: %v\n", err)
		os.Exit(1)
	}

	for _, b := range bindings {
		var selected string
		prompt := binding.ExtractPromptContext(finalCommand, b)

		if b.Type == binding.BindingDirectory {
			// List files and show picker
			files, err := binding.ListFiles(b)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			result := picker.PickString(files, prompt, b.Optional, false)
			if result.Action == picker.ActionCancel {
				os.Exit(0) // User cancelled
			}
			if result.Action == picker.ActionSkip {
				// Remove binding and flag from command
				finalCommand = binding.RemoveWithFlag(finalCommand, b)
				continue
			}
			// Use absolute path
			selected = binding.GetAbsolutePath(b, result.Value)

		} else if b.Type == binding.BindingBooleanFlag {
			// Handle optional boolean flag - ask yes/no to include
			include, ok := picker.PromptYesNo(prompt)
			if !ok {
				os.Exit(0) // User cancelled
			}
			if !include {
				// User chose not to include - remove the flag
				finalCommand = binding.RemoveWithFlag(finalCommand, b)
				continue
			}
			// User chose to include - resolve with empty value (just the flag)
			selected = ""

		} else { // BindingValues
			result := picker.PickString(b.Values, prompt, b.Optional, b.AllowCustom)
			if result.Action == picker.ActionCancel {
				os.Exit(0) // User cancelled
			}
			if result.Action == picker.ActionSkip {
				// Remove binding and flag from command
				finalCommand = binding.RemoveWithFlag(finalCommand, b)
				continue
			}
			selected = result.Value
		}

		finalCommand = binding.Resolve(finalCommand, b, selected)
	}

	// Append extra args if provided
	if extraArgs != "" {
		finalCommand = finalCommand + " " + extraArgs
	}

	// Save to history for 'lz last'
	config.AddHistoryEntry(finalCommand, cmd.Name)

	fmt.Printf("Running: %s\n", finalCommand)
	fmt.Println(strings.Repeat("-", 40))

	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "/bin/sh"
	}

	execCmd := exec.Command(shellPath, "-c", finalCommand)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

func cmdRemove(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: name required")
		fmt.Fprintln(os.Stderr, "Usage: lz remove <name>")
		os.Exit(1)
	}

	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.RemoveCommandByName(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	// Update alias file
	if err := shell.UpdateAliases(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	fmt.Printf("Removed '%s'\n", name)
}

// parseTagsFlag extracts -t or --tags flag from args
// Returns the tags and remaining args
func parseTagsFlag(args []string) ([]string, []string) {
	var tags []string
	var remaining []string

	i := 0
	for i < len(args) {
		if args[i] == "-t" || args[i] == "--tags" {
			if i+1 < len(args) {
				tagStr := args[i+1]
				for _, t := range strings.Split(tagStr, ",") {
					t = strings.TrimSpace(t)
					if t != "" {
						tags = append(tags, t)
					}
				}
				i += 2
				continue
			}
		}
		remaining = append(remaining, args[i])
		i++
	}

	return tags, remaining
}

// formatTags formats tags for display
func formatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return "[" + strings.Join(tags, ", ") + "]"
}

func isValidAliasName(name string) bool {
	if len(name) == 0 {
		return false
	}

	// Must start with letter or underscore
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	// Rest must be alphanumeric or underscore
	for _, c := range name[1:] {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}
