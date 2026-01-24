package shell

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"laziest/internal/binding"
	"laziest/internal/config"
)

const sourceLine = `[ -f "$HOME/.config/laziest/aliases.sh" ] && source "$HOME/.config/laziest/aliases.sh"`

// ShellType represents the type of shell
type ShellType int

const (
	Bash ShellType = iota
	Zsh
)

// GetShellRCPath returns the path to the shell's rc file
func GetShellRCPath(shellType ShellType) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch shellType {
	case Bash:
		return filepath.Join(home, ".bashrc"), nil
	case Zsh:
		return filepath.Join(home, ".zshrc"), nil
	default:
		return "", fmt.Errorf("unsupported shell type")
	}
}

// GetAliasFilePath returns the path to the lz aliases file
func GetAliasFilePath() (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "aliases.sh"), nil
}

// DetectShell returns the current shell type
func DetectShell() ShellType {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return Zsh
	}
	return Bash
}

// GenerateAliases creates alias definitions for all commands
func GenerateAliases(cfg *config.Config) string {
	var sb strings.Builder
	sb.WriteString("# Managed by lz - do not edit manually\n")
	sb.WriteString("# Run 'lz' to manage your command aliases\n\n")

	for _, cmd := range cfg.Commands {
		if binding.HasBindings(cmd.Command) {
			// Commands with bindings invoke lz run for interactive resolution
			sb.WriteString(fmt.Sprintf("alias %s='lz run %s'\n", cmd.Name, cmd.Name))
		} else {
			// Regular alias - escape single quotes in the command
			escaped := strings.ReplaceAll(cmd.Command, "'", "'\\''")
			sb.WriteString(fmt.Sprintf("alias %s='%s'\n", cmd.Name, escaped))
		}
	}

	return sb.String()
}

// UpdateAliases writes all aliases to the alias file
func UpdateAliases(cfg *config.Config) error {
	aliasPath, err := GetAliasFilePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(aliasPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	content := GenerateAliases(cfg)

	if err := os.WriteFile(aliasPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write alias file: %w", err)
	}

	return nil
}

// Init adds the source line to shell rc files (one-time setup)
// Returns the list of shells that were updated
func Init() ([]string, error) {
	shells := []struct {
		shellType ShellType
		name      string
	}{
		{Bash, "bash"},
		{Zsh, "zsh"},
	}

	var updated []string
	var errors []string

	for _, shell := range shells {
		rcPath, err := GetShellRCPath(shell.shellType)
		if err != nil {
			continue
		}

		// Check if rc file exists
		if _, err := os.Stat(rcPath); os.IsNotExist(err) {
			continue
		}

		// Check if source line already exists
		alreadyExists, err := containsSourceLine(rcPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", shell.name, err))
			continue
		}

		if alreadyExists {
			continue
		}

		// Append source line
		if err := appendSourceLine(rcPath); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", shell.name, err))
			continue
		}

		updated = append(updated, rcPath)
	}

	// Also create the alias file if it doesn't exist
	aliasPath, err := GetAliasFilePath()
	if err == nil {
		dir := filepath.Dir(aliasPath)
		os.MkdirAll(dir, 0755)
		if _, err := os.Stat(aliasPath); os.IsNotExist(err) {
			os.WriteFile(aliasPath, []byte("# Managed by lz - do not edit manually\n"), 0644)
		}
	}

	if len(errors) > 0 {
		return updated, fmt.Errorf("some shells failed: %s", strings.Join(errors, "; "))
	}

	return updated, nil
}

// containsSourceLine checks if the rc file already has the lz source line
func containsSourceLine(rcPath string) (bool, error) {
	file, err := os.Open(rcPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Check for our source line or any variation that sources lz aliases
		if strings.Contains(line, ".config/laziest/aliases") {
			return true, nil
		}
	}

	return false, scanner.Err()
}

// appendSourceLine adds the source line to the end of an rc file
func appendSourceLine(rcPath string) error {
	file, err := os.OpenFile(rcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Add newlines before and the source line
	content := fmt.Sprintf("\n# lz aliases\n%s\n", sourceLine)
	_, err = file.WriteString(content)
	return err
}

// ReadFromStdin reads command from piped stdin
func ReadFromStdin() (string, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", fmt.Errorf("no input piped to stdin")
	}

	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading stdin: %w", err)
	}

	return strings.Join(lines, "\n"), nil
}

// GetShellName returns a human-readable shell name
func GetShellName(shellType ShellType) string {
	switch shellType {
	case Bash:
		return "bash"
	case Zsh:
		return "zsh"
	default:
		return "unknown"
	}
}
