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
	ActionModify
)

// PickResult represents the result of a picker interaction
type PickResult struct {
	Action     PickAction
	Value      string // Selected value (empty if cancelled/skipped)
	Extra      string // Extra args if ActionSelectWithExtra
	NewName    string // New name if ActionModify
	NewCommand string // New command if ActionModify
	NewTags    string // New tags (comma-separated) if ActionModify
}

// Item represents a selectable item in the picker
type Item struct {
	Name    string
	Command string
}

// filterItems returns indices of items matching the filter text (case-insensitive)
// Matches against name, command, and tags
func filterItems(items []Item, filter string) []int {
	if filter == "" {
		// No filter - return all indices
		indices := make([]int, len(items))
		for i := range items {
			indices[i] = i
		}
		return indices
	}

	filter = strings.ToLower(filter)
	var indices []int

	for i, item := range items {
		// Check name
		if strings.Contains(strings.ToLower(item.Name), filter) {
			indices = append(indices, i)
			continue
		}
		// Check command
		if strings.Contains(strings.ToLower(item.Command), filter) {
			indices = append(indices, i)
			continue
		}
		// Check tags
		for _, tag := range item.Tags {
			if strings.Contains(strings.ToLower(tag), filter) {
				indices = append(indices, i)
				break
			}
		}
	}

	return indices
}

// filterStrings returns indices of strings matching the filter text (case-insensitive)
func filterStrings(items []string, filter string) []int {
	if filter == "" {
		indices := make([]int, len(items))
		for i := range items {
			indices[i] = i
		}
		return indices
	}

	filter = strings.ToLower(filter)
	var indices []int

	for i, item := range items {
		if strings.Contains(strings.ToLower(item), filter) {
			indices = append(indices, i)
		}
	}

	return indices
}

// Pick displays an interactive picker and returns the selected item
// Returns PickResult with action (Cancel, Select, or SelectWithExtra)
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
	for _, item := range items {
		if len(item.Name) > maxNameLen {
			maxNameLen = len(item.Name)
		}
	}

	// Filter state
	filterMode := false
	filterText := ""
	var filteredIndices []int
	prevFilteredCount := len(items) // Track previous filtered count for clearing

	// Initial render
	render(items, selected, maxNameLen, maxTagLen, prompt, "", true, "", nil, len(items))

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

		// Handle delete confirmation mode
		if confirmDelete {
			if buf[0] == 'y' || buf[0] == 'Y' {
				clearLines(prevFilteredCount + 2)
				// Get actual item from filtered index
				actualIdx := selected
				if filteredIndices != nil && len(filteredIndices) > 0 {
					actualIdx = filteredIndices[selected]
				}
				return PickResult{
					Action: ActionDelete,
					Value:  items[actualIdx].Name,
				}
			}
			// Any other key cancels delete
			confirmDelete = false
			render(items, selected, maxNameLen, maxTagLen, prompt, "", false, filterText, filteredIndices, prevFilteredCount)
			continue
		}

		// Handle filter mode input
		if filterMode {
			switch {
			case buf[0] == 27 && n == 1: // Esc - clear filter and exit filter mode
				filterMode = false
				filterText = ""
				filteredIndices = nil
				selected = 0
				render(items, selected, maxNameLen, maxTagLen, prompt, "", false, "", nil, prevFilteredCount+1) // +1 for filter line
				prevFilteredCount = len(items)
				continue

			case buf[0] == 3: // Ctrl+C - cancel picker entirely
				clearLines(prevFilteredCount + 3) // +1 for filter line
				return PickResult{Action: ActionCancel}

			case buf[0] == 13 || buf[0] == 10: // Enter - select current item
				if filteredIndices != nil && len(filteredIndices) > 0 {
					clearLines(len(filteredIndices) + 3) // +1 for filter line
					actualIdx := filteredIndices[selected]
					return PickResult{
						Action: ActionSelect,
						Value:  items[actualIdx].Name,
					}
				}
				// No matches, ignore Enter
				continue

			case buf[0] == 127: // Backspace
				if len(filterText) > 0 {
					filterText = filterText[:len(filterText)-1]
					filteredIndices = filterItems(items, filterText)
					selected = 0
					render(items, selected, maxNameLen, maxTagLen, prompt, "", false, filterText, filteredIndices, prevFilteredCount)
					prevFilteredCount = len(filteredIndices)
					if prevFilteredCount == 0 {
						prevFilteredCount = 1 // for "(no matches)" line
					}
				}
				continue

			case buf[0] >= 32 && buf[0] < 127: // Printable ASCII
				filterText += string(buf[0])
				filteredIndices = filterItems(items, filterText)
				selected = 0
				render(items, selected, maxNameLen, maxTagLen, prompt, "", false, filterText, filteredIndices, prevFilteredCount)
				prevFilteredCount = len(filteredIndices)
				if prevFilteredCount == 0 {
					prevFilteredCount = 1 // for "(no matches)" line
				}
				continue

			case n == 3 && buf[0] == 27 && buf[1] == 91: // Arrow keys in filter mode
				displayCount := len(filteredIndices)
				if displayCount == 0 {
					continue
				}
				switch buf[2] {
				case 65: // Up
					if selected > 0 {
						selected--
						render(items, selected, maxNameLen, maxTagLen, prompt, "", false, filterText, filteredIndices, prevFilteredCount)
					}
				case 66: // Down
					if selected < displayCount-1 {
						selected++
						render(items, selected, maxNameLen, maxTagLen, prompt, "", false, filterText, filteredIndices, prevFilteredCount)
					}
				}
				continue
			}
			continue
		}

		// Handle normal mode input
		switch {
		case buf[0] == 'q', buf[0] == 27 && n == 1: // q or Esc
			clearLines(prevFilteredCount + 2)
			return PickResult{Action: ActionCancel}

		case buf[0] == 3: // Ctrl+C
			clearLines(prevFilteredCount + 2)
			return PickResult{Action: ActionCancel}

		case buf[0] == '/': // Enter filter mode
			filterMode = true
			filterText = ""
			filteredIndices = filterItems(items, "")
			render(items, selected, maxNameLen, maxTagLen, prompt, "", false, filterText, filteredIndices, prevFilteredCount)
			prevFilteredCount = len(items)
			continue

		case buf[0] == 'x', buf[0] == 'X': // x - delete
			confirmDelete = true
			actualIdx := selected
			if filteredIndices != nil && len(filteredIndices) > 0 {
				actualIdx = filteredIndices[selected]
			}
			render(items, selected, maxNameLen, maxTagLen, prompt, fmt.Sprintf("Delete '%s'? (y/n)", items[actualIdx].Name), false, filterText, filteredIndices, prevFilteredCount)

		case buf[0] == 'm', buf[0] == 'M': // m - modify
			actualIdx := selected
			if filteredIndices != nil && len(filteredIndices) > 0 {
				actualIdx = filteredIndices[selected]
			}
			item := items[actualIdx]
			clearLines(prevFilteredCount + 2)

			// Prompt for new name
			newName := PromptInput(fmt.Sprintf("New name [%s]: ", item.Name))
			if newName == "" {
				newName = item.Name
			}

			// Prompt for new command
			newCmd := PromptInput(fmt.Sprintf("New command [%s]: ", item.Command))
			if newCmd == "" {
				newCmd = item.Command
			}

			// Prompt for new tags
			currentTags := strings.Join(item.Tags, ",")
			tagsPrompt := "New tags (comma-separated)"
			if currentTags != "" {
				tagsPrompt = fmt.Sprintf("New tags [%s]", currentTags)
			}
			newTags := PromptInput(tagsPrompt + ": ")
			if newTags == "" {
				newTags = currentTags
			}

			return PickResult{
				Action:     ActionModify,
				Value:      item.Name, // Original name for lookup
				NewName:    newName,
				NewCommand: newCmd,
				NewTags:    newTags,
			}

		case buf[0] == 'e', buf[0] == 'E': // e - extra args
			clearLines(prevFilteredCount + 2)
			extra := PromptInput("Extra arguments: ")
			if extra == "" {
				// User cancelled extra input, go back to picker
				render(items, selected, maxNameLen, maxTagLen, prompt, "", false, filterText, filteredIndices, prevFilteredCount)
				continue
			}
			actualIdx := selected
			if filteredIndices != nil && len(filteredIndices) > 0 {
				actualIdx = filteredIndices[selected]
			}
			return PickResult{
				Action: ActionSelectWithExtra,
				Value:  items[actualIdx].Name,
				Extra:  extra,
			}

		case buf[0] == 13 || buf[0] == 10: // Enter
			clearLines(prevFilteredCount + 2)
			actualIdx := selected
			if filteredIndices != nil && len(filteredIndices) > 0 {
				actualIdx = filteredIndices[selected]
			}
			return PickResult{
				Action: ActionSelect,
				Value:  items[actualIdx].Name,
			}

		case buf[0] == 'k', buf[0] == 'K': // k - up
			if selected > 0 {
				selected--
				render(items, selected, maxNameLen, maxTagLen, prompt, "", false, filterText, filteredIndices, prevFilteredCount)
			}

		case buf[0] == 'j', buf[0] == 'J': // j - down
			displayCount := len(items)
			if filteredIndices != nil {
				displayCount = len(filteredIndices)
			}
			if selected < displayCount-1 {
				selected++
				render(items, selected, maxNameLen, maxTagLen, prompt, "", false, filterText, filteredIndices, prevFilteredCount)
			}

		case n == 3 && buf[0] == 27 && buf[1] == 91: // Arrow keys
			displayCount := len(items)
			if filteredIndices != nil {
				displayCount = len(filteredIndices)
			}
			switch buf[2] {
			case 65: // Up
				if selected > 0 {
					selected--
					render(items, selected, maxNameLen, maxTagLen, prompt, "", false, filterText, filteredIndices, prevFilteredCount)
				}
			case 66: // Down
				if selected < displayCount-1 {
					selected++
					render(items, selected, maxNameLen, maxTagLen, prompt, "", false, filterText, filteredIndices, prevFilteredCount)
				}
			}
		}
	}
}

// getTerminalWidth returns the terminal width, defaulting to 80 if it can't be determined
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80 // Default fallback
	}
	return width
}

// truncateString truncates a string to maxLen, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if maxLen <= 3 {
		return s[:min(len(s), maxLen)]
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// render draws the picker UI
// confirmMsg is shown instead of help line when non-empty (for delete confirmation)
// firstRender should be true on the initial render to skip clearing non-existent lines
// filterText is the current filter (empty if not filtering)
// filteredIndices contains indices into items of matching items (nil means show all)
// totalItems is the total count of items (for clearing correct number of lines when filtered)
func render(items []Item, selected int, maxNameLen int, maxTagLen int, prompt string, confirmMsg string, firstRender bool, filterText string, filteredIndices []int, totalItems int) {
	// Determine how many lines to clear
	// When filtering, we need to clear based on what was previously rendered
	linesToClear := totalItems + 2
	if filterText != "" {
		linesToClear = totalItems + 3 // +1 for filter line
	}

	// Move cursor to start and clear (skip on first render - nothing to clear yet)
	if !firstRender {
		clearLines(linesToClear)
	}

	// Print prompt
	fmt.Printf("%s\r\n", prompt)

	// Determine which items to display
	var displayIndices []int
	if filteredIndices != nil {
		displayIndices = filteredIndices
	} else {
		displayIndices = make([]int, len(items))
		for i := range items {
			displayIndices[i] = i
		}
	}

	// Get terminal width for truncation
	termWidth := getTerminalWidth()
	// Calculate max command width: termWidth - prefix - name - tags - spacing
	// Prefix: "  > " (4) or "    " (4), spacing between columns: "  " (2) + "  " (2)
	maxCmdWidth := termWidth - 4 - maxNameLen - 2 - maxTagLen - 2 - 1 // -1 for safety margin
	if maxCmdWidth < 20 {
		maxCmdWidth = 20 // Minimum command width
	}

	// Print items with tags
	if len(displayIndices) == 0 {
		fmt.Printf("  \033[2m(no matches)\033[0m\r\n")
	} else {
		for i, idx := range displayIndices {
			item := items[idx]
			tagStr := formatTagsDisplay(item.Tags)
			cmdDisplay := truncateString(item.Command, maxCmdWidth)
			if i == selected {
				fmt.Printf("  \033[7m> %-*s  %-*s  %s\033[0m\r\n", maxNameLen, item.Name, maxTagLen, tagStr, cmdDisplay)
			} else {
				fmt.Printf("    %-*s  %-*s  %s\r\n", maxNameLen, item.Name, maxTagLen, tagStr, cmdDisplay)
			}
		}
	}

	// Print filter line if filtering
	if filterText != "" {
		fmt.Printf("  \033[36m/%s\033[0m\r\n", filterText) // Cyan color for filter
	}

	// Print help or confirmation message
	if confirmMsg != "" {
		fmt.Printf("\033[33m  %s\033[0m", confirmMsg) // Yellow color for confirmation
	} else if filterText != "" {
		fmt.Printf("\033[2m  [↑/↓] navigate  [Enter] select  [Esc] clear filter  [Ctrl+C] cancel\033[0m")
	} else {
		fmt.Printf("\033[2m  [↑/↓/j/k] navigate  [Enter] select  [/] filter  [e] extra  [m] modify  [x] delete  [q] cancel\033[0m")
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

	// Filter state
	filterMode := false
	filterText := ""
	var filteredIndices []int
	prevFilteredCount := len(displayItems)

	// Initial render
	renderStrings(displayItems, selected, prompt, optional, allowCustom, true, "", nil, len(displayItems))

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

		// Handle filter mode input
		if filterMode {
			switch {
			case buf[0] == 27 && n == 1: // Esc - clear filter and exit filter mode
				filterMode = false
				filterText = ""
				filteredIndices = nil
				selected = 0
				renderStrings(displayItems, selected, prompt, optional, allowCustom, false, "", nil, prevFilteredCount+1)
				prevFilteredCount = len(displayItems)
				continue

			case buf[0] == 3: // Ctrl+C - cancel picker entirely
				clearLines(prevFilteredCount + 3)
				return PickResult{Action: ActionCancel}

			case buf[0] == 13 || buf[0] == 10: // Enter - select current item
				if filteredIndices != nil && len(filteredIndices) > 0 {
					clearLines(len(filteredIndices) + 3)
					actualIdx := filteredIndices[selected]
					// Check if [Skip] was selected
					if optional && actualIdx == 0 {
						return PickResult{Action: ActionSkip}
					}
					// Check if [Custom] was selected
					if allowCustom && actualIdx == len(displayItems)-1 {
						value := PromptInput(prompt + " ")
						if value == "" {
							renderStrings(displayItems, selected, prompt, optional, allowCustom, false, filterText, filteredIndices, prevFilteredCount)
							continue
						}
						return PickResult{Action: ActionCustom, Value: value}
					}
					// Return the actual item
					itemIdx := actualIdx - skipOffset
					return PickResult{
						Action: ActionSelect,
						Value:  items[itemIdx],
					}
				}
				continue

			case buf[0] == 127: // Backspace
				if len(filterText) > 0 {
					filterText = filterText[:len(filterText)-1]
					filteredIndices = filterStrings(displayItems, filterText)
					selected = 0
					renderStrings(displayItems, selected, prompt, optional, allowCustom, false, filterText, filteredIndices, prevFilteredCount)
					prevFilteredCount = len(filteredIndices)
					if prevFilteredCount == 0 {
						prevFilteredCount = 1
					}
				}
				continue

			case buf[0] >= 32 && buf[0] < 127: // Printable ASCII
				filterText += string(buf[0])
				filteredIndices = filterStrings(displayItems, filterText)
				selected = 0
				renderStrings(displayItems, selected, prompt, optional, allowCustom, false, filterText, filteredIndices, prevFilteredCount)
				prevFilteredCount = len(filteredIndices)
				if prevFilteredCount == 0 {
					prevFilteredCount = 1
				}
				continue

			case n == 3 && buf[0] == 27 && buf[1] == 91: // Arrow keys
				displayCount := len(filteredIndices)
				if displayCount == 0 {
					continue
				}
				switch buf[2] {
				case 65: // Up
					if selected > 0 {
						selected--
						renderStrings(displayItems, selected, prompt, optional, allowCustom, false, filterText, filteredIndices, prevFilteredCount)
					}
				case 66: // Down
					if selected < displayCount-1 {
						selected++
						renderStrings(displayItems, selected, prompt, optional, allowCustom, false, filterText, filteredIndices, prevFilteredCount)
					}
				}
				continue
			}
			continue
		}

		// Handle normal mode input
		switch {
		case buf[0] == 'q', buf[0] == 27 && n == 1: // q or Esc
			clearLines(prevFilteredCount + 2)
			return PickResult{Action: ActionCancel}

		case buf[0] == 3: // Ctrl+C
			clearLines(prevFilteredCount + 2)
			return PickResult{Action: ActionCancel}

		case buf[0] == '/': // Enter filter mode
			filterMode = true
			filterText = ""
			filteredIndices = filterStrings(displayItems, "")
			renderStrings(displayItems, selected, prompt, optional, allowCustom, false, filterText, filteredIndices, prevFilteredCount)
			prevFilteredCount = len(displayItems)
			continue

		case buf[0] == 's', buf[0] == 'S': // s - skip (only for optional)
			if optional {
				clearLines(prevFilteredCount + 2)
				return PickResult{Action: ActionSkip}
			}

		case buf[0] == 'c', buf[0] == 'C': // c - custom input (only if allowCustom)
			if allowCustom {
				clearLines(prevFilteredCount + 2)
				value := PromptInput(prompt + " ")
				if value == "" {
					// User cancelled, go back to picker
					renderStrings(displayItems, selected, prompt, optional, allowCustom, false, "", nil, len(displayItems))
					continue
				}
				return PickResult{Action: ActionCustom, Value: value}
			}

		case buf[0] == 13 || buf[0] == 10: // Enter
			clearLines(prevFilteredCount + 2)
			// Check if [Skip] was selected
			if optional && selected == 0 {
				return PickResult{Action: ActionSkip}
			}
			// Check if [Custom] was selected
			if allowCustom && selected == len(displayItems)-1 {
				value := PromptInput(prompt + " ")
				if value == "" {
					// User cancelled, go back to picker
					renderStrings(displayItems, selected, prompt, optional, allowCustom, false, "", nil, len(displayItems))
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
				renderStrings(displayItems, selected, prompt, optional, allowCustom, false, "", nil, prevFilteredCount)
			}

		case buf[0] == 'j', buf[0] == 'J': // j - down
			if selected < len(displayItems)-1 {
				selected++
				renderStrings(displayItems, selected, prompt, optional, allowCustom, false, "", nil, prevFilteredCount)
			}

		case n == 3 && buf[0] == 27 && buf[1] == 91: // Arrow keys
			switch buf[2] {
			case 65: // Up
				if selected > 0 {
					selected--
					renderStrings(displayItems, selected, prompt, optional, allowCustom, false, "", nil, prevFilteredCount)
				}
			case 66: // Down
				if selected < len(displayItems)-1 {
					selected++
					renderStrings(displayItems, selected, prompt, optional, allowCustom, false, "", nil, prevFilteredCount)
				}
			}
		}
	}
}

// renderStrings draws the picker UI for string items
// firstRender should be true on the initial render to skip clearing non-existent lines
// filterText is the current filter (empty if not filtering)
// filteredIndices contains indices into items of matching items (nil means show all)
// totalItems is the total count for clearing
func renderStrings(items []string, selected int, prompt string, optional bool, allowCustom bool, firstRender bool, filterText string, filteredIndices []int, totalItems int) {
	// Determine how many lines to clear
	linesToClear := totalItems + 2
	if filterText != "" {
		linesToClear = totalItems + 3 // +1 for filter line
	}

	// Move cursor to start and clear (skip on first render - nothing to clear yet)
	if !firstRender {
		clearLines(linesToClear)
	}

	// Print prompt
	fmt.Printf("%s\r\n", prompt)

	// Determine which items to display
	var displayIndices []int
	if filteredIndices != nil {
		displayIndices = filteredIndices
	} else {
		displayIndices = make([]int, len(items))
		for i := range items {
			displayIndices[i] = i
		}
	}

	// Print items
	if len(displayIndices) == 0 {
		fmt.Printf("  \033[2m(no matches)\033[0m\r\n")
	} else {
		for i, idx := range displayIndices {
			item := items[idx]
			if i == selected {
				fmt.Printf("  \033[7m> %s\033[0m\r\n", item)
			} else {
				fmt.Printf("    %s\r\n", item)
			}
		}
	}

	// Print filter line if filtering
	if filterText != "" {
		fmt.Printf("  \033[36m/%s\033[0m\r\n", filterText) // Cyan color for filter
	}

	// Build help line based on available options
	if filterText != "" {
		fmt.Printf("\033[2m  [↑/↓] navigate  [Enter] select  [Esc] clear filter  [Ctrl+C] cancel\033[0m")
	} else {
		helpParts := []string{"[↑/↓/j/k] navigate", "[Enter] select", "[/] filter"}
		if allowCustom {
			helpParts = append(helpParts, "[c] custom")
		}
		if optional {
			helpParts = append(helpParts, "[s] skip")
		}
		helpParts = append(helpParts, "[q/Esc] cancel")
		fmt.Printf("\033[2m  %s\033[0m", strings.Join(helpParts, "  "))
	}
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
			fmt.Print("\r\n")
			return ""

		case buf[0] == 3: // Ctrl+C
			fmt.Print("\r\n")
			return ""

		case buf[0] == 13 || buf[0] == 10: // Enter
			fmt.Print("\r\n")
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
