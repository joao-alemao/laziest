package picker

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// PickAction represents the action taken in the picker
type PickAction int

const (
	ActionCancel PickAction = iota
	ActionSelect
	ActionSelectWithExtra
	ActionSkip
	ActionCustom
	ActionDelete
)

// PickResult represents the result of a picker interaction
type PickResult struct {
	Action PickAction
	Value  string // Selected value (empty if cancelled/skipped)
	Extra  string // Extra args if ActionSelectWithExtra
}

// Item represents a selectable item in the picker
type Item struct {
	Name    string
	Command string
	Tags    []string
}

// formatTagsDisplay formats tags for picker display
func formatTagsDisplay(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	return "[" + strings.Join(tags, ", ") + "]"
}

// Pick displays an interactive picker and returns the selected item
// Returns PickResult with action (Cancel, Select, SelectWithExtra, or Delete)
func Pick(items []Item, prompt string) PickResult {
	if len(items) == 0 {
		return PickResult{Action: ActionCancel}
	}

	// Get terminal file descriptor
	fd := int(os.Stdin.Fd())

	// Check if we're in a terminal
	if !term.IsTerminal(fd) {
		// Not a terminal, can't show interactive picker
		fmt.Fprintln(os.Stderr, "Cannot show interactive picker: not a terminal")
		return PickResult{Action: ActionCancel}
	}

	// Save terminal state and enable raw mode
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable raw mode: %v\n", err)
		return PickResult{Action: ActionCancel}
	}
	defer term.Restore(fd, oldState)

	selected := 0
	maxNameLen := 0
	maxTagLen := 0
	for _, item := range items {
		if len(item.Name) > maxNameLen {
			maxNameLen = len(item.Name)
		}
		tagStr := formatTagsDisplay(item.Tags)
		if len(tagStr) > maxTagLen {
			maxTagLen = len(tagStr)
		}
	}

	// Initial render
	render(items, selected, maxNameLen, maxTagLen, prompt, "")

	// Input loop
	buf := make([]byte, 3)
	confirmDelete := false

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return PickResult{Action: ActionCancel}
		}

		if n == 0 {
			continue
		}

		// Handle delete confirmation mode
		if confirmDelete {
			if buf[0] == 'y' || buf[0] == 'Y' {
				clearLines(len(items) + 2)
				return PickResult{
					Action: ActionDelete,
					Value:  items[selected].Name,
				}
			}
			// Any other key cancels delete
			confirmDelete = false
			render(items, selected, maxNameLen, maxTagLen, prompt, "")
			continue
		}

		// Handle input
		switch {
		case buf[0] == 'q', buf[0] == 27 && n == 1: // q or Esc
			clearLines(len(items) + 2)
			return PickResult{Action: ActionCancel}

		case buf[0] == 3: // Ctrl+C
			clearLines(len(items) + 2)
			return PickResult{Action: ActionCancel}

		case buf[0] == 'x', buf[0] == 'X': // x - delete
			confirmDelete = true
			render(items, selected, maxNameLen, maxTagLen, prompt, fmt.Sprintf("Delete '%s'? (y/n)", items[selected].Name))

		case buf[0] == 'e', buf[0] == 'E': // e - extra args
			clearLines(len(items) + 2)
			extra := PromptInput("Extra arguments: ")
			if extra == "" {
				// User cancelled extra input, go back to picker
				render(items, selected, maxNameLen, maxTagLen, prompt, "")
				continue
			}
			return PickResult{
				Action: ActionSelectWithExtra,
				Value:  items[selected].Name,
				Extra:  extra,
			}

		case buf[0] == 13 || buf[0] == 10: // Enter
			clearLines(len(items) + 2)
			return PickResult{
				Action: ActionSelect,
				Value:  items[selected].Name,
			}

		case buf[0] == 'k', buf[0] == 'K': // k - up
			if selected > 0 {
				selected--
				render(items, selected, maxNameLen, maxTagLen, prompt, "")
			}

		case buf[0] == 'j', buf[0] == 'J': // j - down
			if selected < len(items)-1 {
				selected++
				render(items, selected, maxNameLen, maxTagLen, prompt, "")
			}

		case n == 3 && buf[0] == 27 && buf[1] == 91: // Arrow keys
			switch buf[2] {
			case 65: // Up
				if selected > 0 {
					selected--
					render(items, selected, maxNameLen, maxTagLen, prompt, "")
				}
			case 66: // Down
				if selected < len(items)-1 {
					selected++
					render(items, selected, maxNameLen, maxTagLen, prompt, "")
				}
			}
		}
	}
}

// render draws the picker UI
// confirmMsg is shown instead of help line when non-empty (for delete confirmation)
func render(items []Item, selected int, maxNameLen int, maxTagLen int, prompt string, confirmMsg string) {
	// Move cursor to start and clear
	clearLines(len(items) + 2)

	// Print prompt
	fmt.Printf("%s\r\n", prompt)

	// Print items with tags
	for i, item := range items {
		tagStr := formatTagsDisplay(item.Tags)
		if i == selected {
			fmt.Printf("  \033[7m> %-*s  %-*s  %s\033[0m\r\n", maxNameLen, item.Name, maxTagLen, tagStr, item.Command)
		} else {
			fmt.Printf("    %-*s  %-*s  %s\r\n", maxNameLen, item.Name, maxTagLen, tagStr, item.Command)
		}
	}

	// Print help or confirmation message
	if confirmMsg != "" {
		fmt.Printf("\033[33m  %s\033[0m", confirmMsg) // Yellow color for confirmation
	} else {
		fmt.Printf("\033[2m  [↑/↓/j/k] navigate  [Enter] select  [e] extra args  [x] delete  [q/Esc] cancel\033[0m")
	}
}

// clearLines moves cursor up and clears lines
func clearLines(n int) {
	for i := 0; i < n; i++ {
		fmt.Print("\033[2K") // Clear line
		if i < n-1 {
			fmt.Print("\033[A") // Move up
		}
	}
	fmt.Print("\r") // Move to start of line
}

// PickString displays an interactive picker for a list of strings
// Returns PickResult with action (Cancel, Select, Skip, or Custom)
func PickString(items []string, prompt string, optional bool, allowCustom bool) PickResult {
	// If allowCustom with no predefined values, go straight to input
	if allowCustom && len(items) == 0 {
		value := PromptInput(prompt + " ")
		if value == "" {
			if optional {
				return PickResult{Action: ActionSkip}
			}
			return PickResult{Action: ActionCancel}
		}
		return PickResult{Action: ActionCustom, Value: value}
	}

	if len(items) == 0 {
		return PickResult{Action: ActionCancel}
	}

	// Build display items with special options
	displayItems := make([]string, 0, len(items)+2)
	skipOffset := 0

	if optional {
		displayItems = append(displayItems, "[Skip]")
		skipOffset = 1
	}

	displayItems = append(displayItems, items...)

	if allowCustom {
		displayItems = append(displayItems, "[Custom]")
	}

	// Get terminal file descriptor
	fd := int(os.Stdin.Fd())

	// Check if we're in a terminal
	if !term.IsTerminal(fd) {
		fmt.Fprintln(os.Stderr, "Cannot show interactive picker: not a terminal")
		return PickResult{Action: ActionCancel}
	}

	// Save terminal state and enable raw mode
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable raw mode: %v\n", err)
		return PickResult{Action: ActionCancel}
	}
	defer term.Restore(fd, oldState)

	selected := 0

	// Initial render
	renderStrings(displayItems, selected, prompt, optional, allowCustom)

	// Input loop
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return PickResult{Action: ActionCancel}
		}

		if n == 0 {
			continue
		}

		// Handle input
		switch {
		case buf[0] == 'q', buf[0] == 27 && n == 1: // q or Esc
			clearLines(len(displayItems) + 2)
			return PickResult{Action: ActionCancel}

		case buf[0] == 3: // Ctrl+C
			clearLines(len(displayItems) + 2)
			return PickResult{Action: ActionCancel}

		case buf[0] == 's', buf[0] == 'S': // s - skip (only for optional)
			if optional {
				clearLines(len(displayItems) + 2)
				return PickResult{Action: ActionSkip}
			}

		case buf[0] == 'c', buf[0] == 'C': // c - custom input (only if allowCustom)
			if allowCustom {
				clearLines(len(displayItems) + 2)
				value := PromptInput(prompt + " ")
				if value == "" {
					// User cancelled, go back to picker
					renderStrings(displayItems, selected, prompt, optional, allowCustom)
					continue
				}
				return PickResult{Action: ActionCustom, Value: value}
			}

		case buf[0] == 13 || buf[0] == 10: // Enter
			clearLines(len(displayItems) + 2)
			// Check if [Skip] was selected
			if optional && selected == 0 {
				return PickResult{Action: ActionSkip}
			}
			// Check if [Custom] was selected
			if allowCustom && selected == len(displayItems)-1 {
				value := PromptInput(prompt + " ")
				if value == "" {
					// User cancelled, go back to picker
					renderStrings(displayItems, selected, prompt, optional, allowCustom)
					continue
				}
				return PickResult{Action: ActionCustom, Value: value}
			}
			// Return the actual item (accounting for skip offset)
			actualIndex := selected - skipOffset
			return PickResult{
				Action: ActionSelect,
				Value:  items[actualIndex],
			}

		case buf[0] == 'k', buf[0] == 'K': // k - up
			if selected > 0 {
				selected--
				renderStrings(displayItems, selected, prompt, optional, allowCustom)
			}

		case buf[0] == 'j', buf[0] == 'J': // j - down
			if selected < len(displayItems)-1 {
				selected++
				renderStrings(displayItems, selected, prompt, optional, allowCustom)
			}

		case n == 3 && buf[0] == 27 && buf[1] == 91: // Arrow keys
			switch buf[2] {
			case 65: // Up
				if selected > 0 {
					selected--
					renderStrings(displayItems, selected, prompt, optional, allowCustom)
				}
			case 66: // Down
				if selected < len(displayItems)-1 {
					selected++
					renderStrings(displayItems, selected, prompt, optional, allowCustom)
				}
			}
		}
	}
}

// renderStrings draws the picker UI for string items
func renderStrings(items []string, selected int, prompt string, optional bool, allowCustom bool) {
	// Move cursor to start and clear
	clearLines(len(items) + 2)

	// Print prompt
	fmt.Printf("%s\r\n", prompt)

	// Print items
	for i, item := range items {
		if i == selected {
			fmt.Printf("  \033[7m> %s\033[0m\r\n", item)
		} else {
			fmt.Printf("    %s\r\n", item)
		}
	}

	// Build help line based on available options
	helpParts := []string{"[↑/↓/j/k] navigate", "[Enter] select"}
	if allowCustom {
		helpParts = append(helpParts, "[c] custom")
	}
	if optional {
		helpParts = append(helpParts, "[s] skip")
	}
	helpParts = append(helpParts, "[q/Esc] cancel")

	fmt.Printf("\033[2m  %s\033[0m", strings.Join(helpParts, "  "))
}

// PromptInput displays a simple inline input prompt and returns the user's input
// Returns empty string if user cancels (Esc or Ctrl+C)
func PromptInput(prompt string) string {
	fd := int(os.Stdin.Fd())

	// Check if we're in a terminal
	if !term.IsTerminal(fd) {
		fmt.Fprintln(os.Stderr, "Cannot show input prompt: not a terminal")
		return ""
	}

	// Save terminal state and enable raw mode
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable raw mode: %v\n", err)
		return ""
	}
	defer term.Restore(fd, oldState)

	fmt.Printf("%s", prompt)

	var input []rune
	buf := make([]byte, 3)

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			fmt.Println()
			return ""
		}

		if n == 0 {
			continue
		}

		switch {
		case buf[0] == 27 && n == 1: // Esc
			fmt.Println()
			return ""

		case buf[0] == 3: // Ctrl+C
			fmt.Println()
			return ""

		case buf[0] == 13 || buf[0] == 10: // Enter
			fmt.Println()
			return string(input)

		case buf[0] == 127: // Backspace
			if len(input) > 0 {
				input = input[:len(input)-1]
				// Clear line and reprint
				fmt.Print("\r\033[K")
				fmt.Printf("%s%s", prompt, string(input))
			}

		case buf[0] >= 32 && buf[0] < 127: // Printable ASCII
			input = append(input, rune(buf[0]))
			fmt.Printf("%c", buf[0])
		}
	}
}
