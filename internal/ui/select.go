package ui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// Select presents an interactive arrow-key picker and returns the chosen option.
// Returns an error if the list is empty or the user presses Ctrl+C.
func Select(label string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options to select from")
	}

	selected, err := runSelect(label, options, -1, false)
	if err != nil {
		return "", err
	}
	return options[selected], nil
}

// SelectWithDefault is like Select but pre-selects the option matching defaultVal.
func SelectWithDefault(label string, options []string, defaultVal string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options to select from")
	}

	defaultIdx := -1
	for i, o := range options {
		if strings.EqualFold(o, defaultVal) {
			defaultIdx = i
			break
		}
	}

	selected, err := runSelect(label, options, defaultIdx, false)
	if err != nil {
		return "", err
	}
	return options[selected], nil
}

// SelectOrCreate presents a picker with a "+ Create new" option at the top.
// Returns (value, isNew, error). If the user picks "Create new", isNew is true
// and value is empty (caller should prompt for the new value).
func SelectOrCreate(label string, options []string, createLabel string) (string, bool, error) {
	if createLabel == "" {
		createLabel = "+ Create new"
	}

	display := make([]string, 0, len(options)+1)
	display = append(display, createLabel)
	display = append(display, options...)

	selected, err := runSelect(label, display, -1, true)
	if err != nil {
		return "", false, err
	}

	if selected == 0 {
		return "", true, nil
	}
	return options[selected-1], false, nil
}

// filteredItem maps a filtered display index back to the original options index.
type filteredItem struct {
	origIdx int
	text    string
}

func filterOptions(options []string, query string, pinnedIdx int) []filteredItem {
	if query == "" {
		result := make([]filteredItem, len(options))
		for i, o := range options {
			result[i] = filteredItem{origIdx: i, text: o}
		}
		return result
	}

	q := strings.ToLower(query)
	var result []filteredItem

	// Always include pinned item (e.g. "+ Create new") if it exists
	if pinnedIdx >= 0 && pinnedIdx < len(options) {
		result = append(result, filteredItem{origIdx: pinnedIdx, text: options[pinnedIdx]})
	}

	for i, o := range options {
		if i == pinnedIdx {
			continue // already added
		}
		if strings.Contains(strings.ToLower(o), q) {
			result = append(result, filteredItem{origIdx: i, text: o})
		}
	}
	return result
}

// runSelect is the core interactive selector with type-to-filter search.
// If hasCreateOption is true, the first option is pinned (always visible).
func runSelect(label string, options []string, defaultIdx int, hasCreateOption bool) (int, error) {
	// Fall back to simple prompt if stdin is not a terminal.
	if !isTerminal(int(os.Stdin.Fd())) {
		return fallbackSelect(label, options, defaultIdx)
	}

	fd := int(os.Stdin.Fd())
	oldState, err := makeRaw(fd)
	if err != nil {
		return fallbackSelect(label, options, defaultIdx)
	}
	defer restoreTerminal(fd, oldState)

	query := ""
	cursor := 0
	pinnedIdx := -1
	if hasCreateOption {
		pinnedIdx = 0
	}

	filtered := filterOptions(options, query, pinnedIdx)

	if defaultIdx >= 0 && defaultIdx < len(options) {
		// Find the default in the filtered list
		for i, f := range filtered {
			if f.origIdx == defaultIdx {
				cursor = i
				break
			}
		}
	}

	// Print label
	hint := "type to filter, enter to select"
	if len(options) <= 10 {
		hint = "arrows to move, enter to select"
	}
	fmt.Printf("  %s%s%s %s(%s)%s\n", Cyan, label, Reset, Dim, hint, Reset)

	renderFiltered(filtered, cursor, query)

	buf := make([]byte, 3)
	for {
		n, readErr := os.Stdin.Read(buf)
		if readErr != nil {
			return 0, readErr
		}

		prevLines := renderedLines(filtered, query)

		if n == 1 {
			switch {
			case buf[0] == 13 || buf[0] == 10: // Enter
				if len(filtered) > 0 {
					clearRendered(prevLines)
					renderFinal(filtered[cursor].text)
					return filtered[cursor].origIdx, nil
				}
			case buf[0] == 3: // Ctrl+C
				clearRendered(prevLines)
				return 0, fmt.Errorf("cancelled")
			case buf[0] == 27: // Escape — clear filter
				if query != "" {
					query = ""
					filtered = filterOptions(options, query, pinnedIdx)
					cursor = 0
				}
			case buf[0] == 127 || buf[0] == 8: // Backspace
				if len(query) > 0 {
					query = query[:len(query)-1]
					filtered = filterOptions(options, query, pinnedIdx)
					if cursor >= len(filtered) {
						cursor = max(0, len(filtered)-1)
					}
				}
			case buf[0] >= 32 && buf[0] < 127: // Printable char
				query += string(buf[0])
				filtered = filterOptions(options, query, pinnedIdx)
				if cursor >= len(filtered) {
					cursor = max(0, len(filtered)-1)
				}
			}
		} else if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 65: // Up arrow
				if cursor > 0 {
					cursor--
				}
			case 66: // Down arrow
				if cursor < len(filtered)-1 {
					cursor++
				}
			}
		}

		clearRendered(prevLines)
		renderFiltered(filtered, cursor, query)
	}
}

func renderedLines(filtered []filteredItem, query string) int {
	n := len(filtered)
	if query != "" {
		n++ // search line
	}
	if len(filtered) == 0 && query != "" {
		n++ // "no matches" line
	}
	return n
}

func renderFiltered(filtered []filteredItem, cursor int, query string) {
	if query != "" {
		fmt.Printf("  %s/%s%s\n", Dim, query, Reset)
	}

	if len(filtered) == 0 {
		if query != "" {
			fmt.Printf("  %s(no matches)%s\n", Dim, Reset)
		}
		return
	}

	for i, item := range filtered {
		if i == cursor {
			fmt.Printf("  %s▸ %s%s\n", Cyan, item.text, Reset)
		} else {
			fmt.Printf("    %s%s%s\n", Dim, item.text, Reset)
		}
	}
}

func renderFinal(text string) {
	fmt.Printf("  %s▸ %s%s\n", Green, text, Reset)
}

// clearRendered moves up N lines and clears them.
func clearRendered(n int) {
	for i := 0; i < n; i++ {
		fmt.Print("\033[A")  // move up
		fmt.Print("\033[2K") // clear line
	}
	fmt.Print("\r")
}

func isTerminal(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	return err == nil
}

func makeRaw(fd int) (*unix.Termios, error) {
	oldState, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return nil, err
	}

	newState := *oldState
	// Disable canonical mode and echo.
	newState.Lflag &^= unix.ICANON | unix.ECHO
	// Read one byte at a time, no timeout.
	newState.Cc[unix.VMIN] = 1
	newState.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, &newState); err != nil {
		return nil, err
	}
	return oldState, nil
}

func restoreTerminal(fd int, state *unix.Termios) {
	unix.IoctlSetTermios(fd, ioctlWriteTermios, state)
}

// fallbackSelect is used when stdin is not a terminal (e.g. in tests or pipes).
func fallbackSelect(label string, options []string, defaultIdx int) (int, error) {
	fmt.Printf("  %s%s%s:\n", Cyan, label, Reset)
	for i, opt := range options {
		fmt.Printf("    %d) %s\n", i+1, opt)
	}
	def := ""
	if defaultIdx >= 0 {
		def = fmt.Sprintf("%d", defaultIdx+1)
	}
	choice := Prompt("Choice", def)
	var idx int
	if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 || idx > len(options) {
		return 0, fmt.Errorf("invalid choice: %s", choice)
	}
	return idx - 1, nil
}
