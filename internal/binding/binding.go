package binding

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// BindingType represents the type of dynamic binding
type BindingType int

const (
	BindingDirectory BindingType = iota
	BindingValues
)

// Binding represents a dynamic placeholder in a command
type Binding struct {
	Type        BindingType
	Path        string   // For directory bindings (absolute path)
	Filter      string   // Glob filter for directory bindings (e.g., "*.yaml")
	Values      []string // For value bindings
	Placeholder string   // The original placeholder text e.g. "{%/configs:*.yaml%}"
	Optional    bool     // True if binding starts with ? (e.g., {%?...%})
	Flag        string   // Optional flag prefix (e.g., "--debug" from {%--debug:[...]%})
	AllowCustom bool     // True if binding allows custom input (has ... in values)
}

// bindingPattern matches {%...%} placeholders
var bindingPattern = regexp.MustCompile(`\{%(.+?)%\}`)

// Parse extracts all bindings from a command string
// Returns bindings in order of appearance
func Parse(command string) ([]Binding, error) {
	matches := bindingPattern.FindAllStringSubmatchIndex(command, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	var bindings []Binding

	for _, match := range matches {
		// match[0]:match[1] is the full match {%...%}
		// match[2]:match[3] is the inner content
		placeholder := command[match[0]:match[1]]
		content := command[match[2]:match[3]]

		binding, err := parseContent(content, placeholder)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}

	return bindings, nil
}

// parseContent parses the inner content of a binding
func parseContent(content, placeholder string) (Binding, error) {
	content = strings.TrimSpace(content)

	if content == "" {
		return Binding{}, fmt.Errorf("empty binding: %s", placeholder)
	}

	optional := false
	flag := ""

	// Check for optional prefix: ?
	if strings.HasPrefix(content, "?") {
		optional = true
		content = strings.TrimSpace(content[1:])
		if content == "" {
			return Binding{}, fmt.Errorf("empty binding after ?: %s", placeholder)
		}
	}

	// Check for flag prefix: --flag: or -f:
	// Flag must come before [ or /
	flagPattern := regexp.MustCompile(`^(-{1,2}[\w-]+):\s*`)
	if match := flagPattern.FindStringSubmatch(content); match != nil {
		flag = match[1]
		content = strings.TrimSpace(content[len(match[0]):])
		if content == "" {
			return Binding{}, fmt.Errorf("empty binding after flag: %s", placeholder)
		}
	}

	// Check if it's a value binding: [val1,val2,...]
	if strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]") {
		inner := content[1 : len(content)-1]
		if inner == "" {
			return Binding{}, fmt.Errorf("value binding cannot be empty: %s", placeholder)
		}

		values := strings.Split(inner, ",")
		for i, v := range values {
			values[i] = strings.TrimSpace(v)
		}

		// Check for ... (custom input marker) and remove it from values
		allowCustom := false
		var filteredValues []string
		for _, v := range values {
			if v == "..." {
				allowCustom = true
			} else if v == "" {
				return Binding{}, fmt.Errorf("value binding contains empty value: %s", placeholder)
			} else {
				filteredValues = append(filteredValues, v)
			}
		}

		// If only ... was specified, allow custom but no predefined values
		if len(filteredValues) == 0 && !allowCustom {
			return Binding{}, fmt.Errorf("value binding cannot be empty: %s", placeholder)
		}

		return Binding{
			Type:        BindingValues,
			Values:      filteredValues,
			Placeholder: placeholder,
			Optional:    optional,
			Flag:        flag,
			AllowCustom: allowCustom,
		}, nil
	}

	// It's a directory binding: /path or /path:*.ext
	path := content
	filter := ""

	// Check for filter (last colon that's not part of the path)
	// Handle Windows paths (C:\...) by looking for colon not at position 1
	lastColon := strings.LastIndex(content, ":")
	if lastColon > 1 { // Not a Windows drive letter
		path = content[:lastColon]
		filter = content[lastColon+1:]
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// Make path absolute if not already
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err == nil {
			path = absPath
		}
	}

	return Binding{
		Type:        BindingDirectory,
		Path:        path,
		Filter:      filter,
		Placeholder: placeholder,
		Optional:    optional,
		Flag:        flag,
	}, nil
}

// Validate checks if a binding is valid
// Returns warning messages (not errors) for issues that don't prevent adding
func Validate(b Binding) []string {
	var warnings []string

	if b.Type == BindingDirectory {
		info, err := os.Stat(b.Path)
		if os.IsNotExist(err) {
			warnings = append(warnings, fmt.Sprintf("directory '%s' does not exist", b.Path))
		} else if err != nil {
			warnings = append(warnings, fmt.Sprintf("cannot access '%s': %v", b.Path, err))
		} else if !info.IsDir() {
			warnings = append(warnings, fmt.Sprintf("'%s' is not a directory", b.Path))
		}
	}

	return warnings
}

// ListFiles returns files in the binding's directory matching the filter
// Files are returned as relative paths from the directory, sorted alphabetically
// Searches recursively, skips symlinks
func ListFiles(b Binding) ([]string, error) {
	if b.Type != BindingDirectory {
		return nil, fmt.Errorf("ListFiles called on non-directory binding")
	}

	info, err := os.Stat(b.Path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("directory '%s' does not exist", b.Path)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot access '%s': %v", b.Path, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("'%s' is not a directory", b.Path)
	}

	var files []string

	err = filepath.Walk(b.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(b.Path, path)
		if err != nil {
			return nil
		}

		// Apply filter if specified
		if b.Filter != "" {
			matched, err := filepath.Match(b.Filter, info.Name())
			if err != nil || !matched {
				return nil
			}
		}

		files = append(files, relPath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error reading directory '%s': %v", b.Path, err)
	}

	if len(files) == 0 {
		if b.Filter != "" {
			return nil, fmt.Errorf("no files found in '%s' matching '%s'", b.Path, b.Filter)
		}
		return nil, fmt.Errorf("no files found in '%s'", b.Path)
	}

	sort.Strings(files)
	return files, nil
}

// GetAbsolutePath returns the absolute path for a selected relative file
func GetAbsolutePath(b Binding, relativePath string) string {
	return filepath.Join(b.Path, relativePath)
}

// Resolve replaces the binding placeholder with the given value in the command
// If the binding has a flag, it outputs "flag value" (e.g., "--debug True")
func Resolve(command string, b Binding, value string) string {
	replacement := value
	if b.Flag != "" {
		replacement = b.Flag + " " + value
	}
	return strings.Replace(command, b.Placeholder, replacement, 1)
}

// HasBindings checks if a command string contains any bindings
func HasBindings(command string) bool {
	return strings.Contains(command, "{%")
}

// ExtractPromptContext tries to extract context for the picker prompt
// Returns something like "Select file for --config" or "Select value for --env"
func ExtractPromptContext(command string, b Binding) string {
	// If binding has an explicit flag, use it
	if b.Flag != "" {
		if b.Type == BindingDirectory {
			return fmt.Sprintf("Select file for %s [%s]:", b.Flag, b.Path)
		}
		return fmt.Sprintf("Select value for %s:", b.Flag)
	}

	// Otherwise, find the position of this placeholder and look backwards for a flag
	idx := strings.Index(command, b.Placeholder)
	if idx == -1 {
		return defaultPrompt(b)
	}

	// Look backwards for a flag (--something or -x)
	before := command[:idx]
	before = strings.TrimRight(before, " =")

	// Try to find --flag or -f pattern
	flagPattern := regexp.MustCompile(`(-{1,2}[\w-]+)\s*$`)
	match := flagPattern.FindStringSubmatch(before)

	var context string
	if len(match) > 1 {
		context = fmt.Sprintf(" for %s", match[1])
	}

	if b.Type == BindingDirectory {
		return fmt.Sprintf("Select file%s [%s]:", context, b.Path)
	}
	return fmt.Sprintf("Select value%s:", context)
}

func defaultPrompt(b Binding) string {
	if b.Type == BindingDirectory {
		return fmt.Sprintf("Select file [%s]:", b.Path)
	}
	return "Select value:"
}

// RemoveWithFlag removes the binding placeholder and its associated flag from the command
// Used when user skips an optional binding
func RemoveWithFlag(command string, b Binding) string {
	// If binding has a flag, try to remove both flag and placeholder
	if b.Flag != "" {
		// Pattern: flag + optional space/= + placeholder
		// Examples: "--debug {%...%}", "--config={%...%}"
		pattern := regexp.QuoteMeta(b.Flag) + `\s*=?\s*` + regexp.QuoteMeta(b.Placeholder)
		re := regexp.MustCompile(pattern)
		if re.MatchString(command) {
			// Flag appears before placeholder - remove both
			result := re.ReplaceAllString(command, "")
			result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
			return strings.TrimSpace(result)
		}
		// Flag is embedded in placeholder (e.g., {%?--flag:[values]%})
		// Fall through to just remove the placeholder
	}

	// No flag or flag embedded in placeholder - just remove the placeholder
	result := strings.Replace(command, b.Placeholder, "", 1)
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	return strings.TrimSpace(result)
}
