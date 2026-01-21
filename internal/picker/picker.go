package picker

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// Item represents a selectable item in the picker
type Item struct {
	Name    string
	Command string
}

// Pick displays an interactive picker and returns the selected item
// Returns nil if user cancels (q, Esc, or Ctrl+C)
func Pick(items []Item, prompt string) *Item {
	if len(items) == 0 {
		return nil
	}

	if len(items) == 1 {
		return &items[0]
	}

	// Get terminal file descriptor
	fd := int(os.Stdin.Fd())

	// Check if we're in a terminal
	if !term.IsTerminal(fd) {
		// Not a terminal, can't show interactive picker
		fmt.Fprintln(os.Stderr, "Cannot show interactive picker: not a terminal")
		return nil
	}

	// Save terminal state and enable raw mode
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable raw mode: %v\n", err)
		return nil
	}
	defer term.Restore(fd, oldState)

	selected := 0
	maxNameLen := 0
	for _, item := range items {
		if len(item.Name) > maxNameLen {
			maxNameLen = len(item.Name)
		}
	}

	// Initial render
	render(items, selected, maxNameLen, prompt)

	// Input loop
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil
		}

		if n == 0 {
			continue
		}

		// Handle input
		switch {
		case buf[0] == 'q', buf[0] == 27 && n == 1: // q or Esc
			clearLines(len(items) + 2)
			return nil

		case buf[0] == 3: // Ctrl+C
			clearLines(len(items) + 2)
			return nil

		case buf[0] == 13 || buf[0] == 10: // Enter
			clearLines(len(items) + 2)
			return &items[selected]

		case buf[0] == 'k', buf[0] == 'K': // k - up
			if selected > 0 {
				selected--
				render(items, selected, maxNameLen, prompt)
			}

		case buf[0] == 'j', buf[0] == 'J': // j - down
			if selected < len(items)-1 {
				selected++
				render(items, selected, maxNameLen, prompt)
			}

		case n == 3 && buf[0] == 27 && buf[1] == 91: // Arrow keys
			switch buf[2] {
			case 65: // Up
				if selected > 0 {
					selected--
					render(items, selected, maxNameLen, prompt)
				}
			case 66: // Down
				if selected < len(items)-1 {
					selected++
					render(items, selected, maxNameLen, prompt)
				}
			}
		}
	}
}

// render draws the picker UI
func render(items []Item, selected int, maxNameLen int, prompt string) {
	// Move cursor to start and clear
	clearLines(len(items) + 2)

	// Print prompt
	fmt.Printf("%s\r\n", prompt)

	// Print items
	for i, item := range items {
		if i == selected {
			fmt.Printf("  \033[7m> %-*s  %s\033[0m\r\n", maxNameLen, item.Name, item.Command)
		} else {
			fmt.Printf("    %-*s  %s\r\n", maxNameLen, item.Name, item.Command)
		}
	}

	// Print help
	fmt.Printf("\033[2m  [↑/↓/j/k] navigate  [Enter] select  [q/Esc] cancel\033[0m")
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
