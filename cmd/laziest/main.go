package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"laziest/internal/binding"
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
	case "run", "r":
		cmdRun(os.Args[2:])
	case "remove", "rm":
		cmdRemove(os.Args[2:])
	case "tags", "t":
		cmdTags()
	case "init":
		cmdInit()
	case "help", "-h", "--help":
		printUsage()
	case "version", "-v", "--version":
		fmt.Printf("laziest version %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`laziest - Quick command aliases manager

Usage:
  laziest                      Interactive command picker
  laziest list [-t <tag>]      Interactive picker, optionally filter by tag
  laziest add <name> <cmd> [-t <tags>]  Add a new command
  laziest run <name> [--extra <args>]   Run command by name
  laziest run -t <tag> [--extra <args>] Pick and run a command with that tag
  laziest remove <name>        Remove a command
  laziest tags                 List all tags with command counts
  laziest init                 One-time setup: add source line to shell rc
  laziest help                 Show this help
  laziest version              Show version

Adding commands:
  laziest add deploy "kubectl apply -f ." -t DevOps,K8s
  echo "git status" | laziest add gs -t Git

Tags:
  - Comma-separated, no spaces: -t Tag1,Tag2
  - Used for filtering and organizing commands

Dynamic bindings:
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
  Example: laziest run train --extra --verbose --epochs 100

Interactive picker keys:
  ↑/↓ or j/k   Navigate
  Enter        Select and run
  e            Add extra args then run
  c            Enter custom value (when ... in binding)
  s            Skip optional binding
  q or Esc     Cancel

Examples:
  laziest add gs "git status" -t Git
  laziest add train "python train.py --config {%/configs:*.yaml%}" -t ML
  laziest add deploy "kubectl apply --dry-run={%[none,client,server]%}" -t K8s
  laziest add debug "python train.py {%?--debug:[True,False]%}" -t ML
  laziest add epochs "python train.py --epochs {%[10,50,100,...]%}" -t ML
  laziest run gs
  laziest run train --extra --verbose
  laziest run -t ML
  laziest list -t Git
  laziest rm gs`)
}

func cmdInit() {
	updated, err := shell.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	if len(updated) == 0 {
		fmt.Println("laziest is already configured in your shell rc files.")
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

func cmdTags() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	counts := cfg.GetTagCounts()
	if len(counts) == 0 {
		fmt.Println("No tags defined. Add tags with: laziest add <name> <cmd> -t <tags>")
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
		fmt.Println("  1. Run 'laziest init' to set up shell integration")
		fmt.Println("  2. Add commands with 'laziest add <name> <command> -t <tags>'")
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
		fmt.Println("  1. Run 'laziest init' to set up shell integration")
		fmt.Println("  2. Add commands with 'laziest add <name> <command> -t <tags>'")
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

func cmdAdd(args []string) {
	// Parse tags flag
	tags, remaining := parseTagsFlag(args)

	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Error: name required")
		fmt.Fprintln(os.Stderr, "Usage: laziest add <name> <command> [-t <tags>]")
		fmt.Fprintln(os.Stderr, "   or: echo 'command' | laziest add <name> [-t <tags>]")
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
			fmt.Fprintln(os.Stderr, "Usage: laziest add <name> <command> [-t <tags>]")
			fmt.Fprintln(os.Stderr, "   or: echo 'command' | laziest add <name> [-t <tags>]")
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
		fmt.Fprintln(os.Stderr, "No commands saved. Use 'laziest add <name> <command>' to add one.")
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
		fmt.Fprintln(os.Stderr, "Usage: laziest run <name>")
		fmt.Fprintln(os.Stderr, "   or: laziest run -t <tag>")
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
}

func cmdRemove(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: name required")
		fmt.Fprintln(os.Stderr, "Usage: laziest remove <name>")
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
