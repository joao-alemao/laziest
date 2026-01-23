package builder

import (
	"fmt"
	"strings"

	"laziest/internal/flagparse"
	"laziest/internal/picker"
)

// BindingChoice represents user's choice for a flag binding
type BindingChoice int

const (
	ChoiceStatic    BindingChoice = iota // Keep the flag static
	ChoiceBoolean                        // Optional boolean flag (include or skip)
	ChoiceDirectory                      // Directory picker binding
	ChoiceValueList                      // List of predefined values
)

// BuildResult represents the result of the interactive builder
type BuildResult struct {
	Command   string // The final command with bindings
	Cancelled bool   // True if user cancelled
}

// BuildCommand runs the interactive command builder
// Takes an example command and walks through each flag to create bindings
func BuildCommand(command string) BuildResult {
	baseCmd, flags := flagparse.Parse(command)

	if len(flags) == 0 {
		// No flags found - return as-is
		return BuildResult{Command: command, Cancelled: false}
	}

	fmt.Printf("\n\033[1mBuilding command from:\033[0m %s\n\n", command)
	fmt.Printf("\033[2mBase command: %s\033[0m\n", baseCmd)
	fmt.Printf("\033[2mFound %d flag(s) to configure\033[0m\n\n", len(flags))

	// Process each flag
	var parts []string
	parts = append(parts, baseCmd)

	for i, flag := range flags {
		fmt.Printf("\033[1m[%d/%d] Flag: %s\033[0m", i+1, len(flags), flag.Name)
		if flag.Value != "" {
			fmt.Printf(" = %s", flag.Value)
		}
		fmt.Println()

		binding, cancelled := processFlag(flag)
		if cancelled {
			return BuildResult{Cancelled: true}
		}

		if binding != "" {
			parts = append(parts, binding)
		}

		fmt.Println()
	}

	result := strings.Join(parts, " ")
	return BuildResult{Command: result, Cancelled: false}
}

// processFlag interactively processes a single flag
// Returns the binding string and whether user cancelled
func processFlag(flag flagparse.Flag) (string, bool) {
	if flag.IsBoolean {
		return processBooleanFlag(flag)
	}
	return processValueFlag(flag)
}

// processBooleanFlag handles flags that have no value or True/False value
func processBooleanFlag(flag flagparse.Flag) (string, bool) {
	// Case 1: Pure boolean flag (no value, e.g., --verbose)
	if flag.Value == "" {
		options := []string{
			"Keep static (always include this flag)",
			"Make optional (choose to include or skip at runtime)",
		}

		idx := picker.PickOption("How should this flag behave?", options)
		if idx == -1 {
			return "", true // cancelled
		}

		switch idx {
		case 0: // Static
			return flag.Name, false
		case 1: // Optional boolean
			// {%?--verbose%} syntax - just the flag, removed if skipped
			return fmt.Sprintf("{%%?%s%%}", flag.Name), false
		}

		return flag.Name, false
	}

	// Case 2: True/False value flag (e.g., --is_debug_run True)
	options := []string{
		fmt.Sprintf("Keep static (always use %s)", flag.Value),
		"Make dynamic (choose True/False at runtime)",
		"Make optional + dynamic (choose True/False or skip entirely)",
	}

	idx := picker.PickOption("How should this flag behave?", options)
	if idx == -1 {
		return "", true // cancelled
	}

	switch idx {
	case 0: // Static
		return flag.Name + " " + flag.Value, false
	case 1: // Dynamic True/False
		return fmt.Sprintf("{%%%s:[True,False]%%}", flag.Name), false
	case 2: // Optional + Dynamic
		return fmt.Sprintf("{%%?%s:[True,False]%%}", flag.Name), false
	}

	return flag.Name + " " + flag.Value, false
}

// processValueFlag handles flags that have a value
func processValueFlag(flag flagparse.Flag) (string, bool) {
	options := []string{
		"Keep static (always use this value)",
		"Directory picker (browse and select a path)",
		"Value list (choose from predefined options)",
	}

	idx := picker.PickOption("How should this flag's value be set?", options)
	if idx == -1 {
		return "", true // cancelled
	}

	switch idx {
	case 0: // Static
		return flag.Name + " " + flag.Value, false

	case 1: // Directory binding
		return buildDirectoryBinding(flag)

	case 2: // Value list
		return buildValueListBinding(flag)
	}

	return flag.Name + " " + flag.Value, false
}

// buildDirectoryBinding creates a directory picker binding
func buildDirectoryBinding(flag flagparse.Flag) (string, bool) {
	// Ask for base directory
	defaultDir := extractDirectory(flag.Value)
	prompt := fmt.Sprintf("Base directory [%s]: ", defaultDir)
	baseDir := picker.PromptInput(prompt)
	if baseDir == "" {
		baseDir = defaultDir
	}

	// Ask for filter pattern
	filter := picker.PromptInput("Filter pattern (e.g., *.yaml, empty for all): ")

	// Ask if optional
	optional, ok := picker.PromptYesNo("Make this flag optional?")
	if !ok {
		return "", true // cancelled
	}

	// Build binding: {%?--flag:/path[:filter]%} or {%--flag:/path[:filter]%}
	var binding string
	if optional {
		binding = fmt.Sprintf("{%%?%s:%s", flag.Name, baseDir)
	} else {
		binding = fmt.Sprintf("{%%%s:%s", flag.Name, baseDir)
	}

	if filter != "" {
		binding += ":" + filter
	}
	binding += "%}"

	return binding, false
}

// buildValueListBinding creates a value list binding
func buildValueListBinding(flag flagparse.Flag) (string, bool) {
	fmt.Println("\033[2mEnter values one per line. Empty line to finish.\033[0m")
	fmt.Printf("\033[2mTip: Add '...' as the last value to allow custom input at runtime.\033[0m\n")

	// Pre-fill with current value as first suggestion
	var values []string
	if flag.Value != "" {
		fmt.Printf("\033[2mSuggested: %s\033[0m\n", flag.Value)
	}

	for {
		v := picker.PromptInput("Value: ")
		if v == "" {
			break
		}
		values = append(values, v)
	}

	if len(values) == 0 {
		// No values entered, keep static
		return flag.Name + " " + flag.Value, false
	}

	// Ask if optional
	optional, ok := picker.PromptYesNo("Make this flag optional?")
	if !ok {
		return "", true // cancelled
	}

	// Build binding: {%?--flag:[val1,val2,...]%} or {%--flag:[val1,val2,...]%}
	valueList := "[" + strings.Join(values, ",") + "]"
	var binding string
	if optional {
		binding = fmt.Sprintf("{%%?%s:%s%%}", flag.Name, valueList)
	} else {
		binding = fmt.Sprintf("{%%%s:%s%%}", flag.Name, valueList)
	}

	return binding, false
}

// extractDirectory extracts the directory portion from a path
// If the value looks like a file, returns its parent directory
// Otherwise returns the value as-is
func extractDirectory(path string) string {
	if path == "" {
		return "."
	}

	// Check if path has a file extension (likely a file)
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash >= 0 {
		remainder := path[lastSlash+1:]
		if strings.Contains(remainder, ".") {
			// Has extension, return parent directory
			return path[:lastSlash]
		}
	}

	return path
}
