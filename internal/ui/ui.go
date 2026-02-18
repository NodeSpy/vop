// Package ui provides terminal output formatting and interactive prompts.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ANSI color codes
const (
	Red    = "\033[0;31m"
	Green  = "\033[0;32m"
	Yellow = "\033[0;33m"
	Cyan   = "\033[0;36m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Reset  = "\033[0m"
)

// Quiet suppresses all informational output (Info, Warn, Success).
// Error output is never suppressed. Used by credential_process mode.
var Quiet bool

func Info(msg string, args ...any) {
	if Quiet {
		return
	}
	fmt.Printf(Green+">>>"+Reset+" "+msg+"\n", args...)
}

func Warn(msg string, args ...any) {
	if Quiet {
		return
	}
	fmt.Fprintf(os.Stderr, Yellow+">>>"+Reset+" "+msg+"\n", args...)
}

func Error(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, Red+">>>"+Reset+" "+msg+"\n", args...)
}

func Success(msg string, args ...any) {
	if Quiet {
		return
	}
	fmt.Printf(Green+" ok"+Reset+" "+msg+"\n", args...)
}

func Fatal(msg string, args ...any) {
	Error(msg, args...)
	os.Exit(1)
}

// Prompt asks the user for input with an optional default value.
func Prompt(label string, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  "+Cyan+"%s"+Reset+" "+Dim+"[%s]"+Reset+": ", label, defaultVal)
	} else {
		fmt.Printf("  "+Cyan+"%s"+Reset+": ", label)
	}

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

// PromptYN asks a yes/no question. defaultYes controls the default answer.
func PromptYN(label string, defaultYes bool) bool {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}
	fmt.Printf("  "+Cyan+"%s"+Reset+" "+Dim+"%s"+Reset+": ", label, hint)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}
