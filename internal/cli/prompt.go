package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

type terminalPrompter struct {
	scanner *bufio.Scanner
}

func newTerminalPrompter() *terminalPrompter {
	return &terminalPrompter{scanner: bufio.NewScanner(os.Stdin)}
}

func (t *terminalPrompter) Confirm(message string) bool {
	fmt.Print("\033[33m" + message + "\033[0m [Y/n] ")
	if t.scanner.Scan() {
		input := strings.TrimSpace(t.scanner.Text())
		switch strings.ToLower(input) {
		case "", "y", "yes":
			return true
		}
	}
	return false
}

func (t *terminalPrompter) Input(prompt string) string {
	fmt.Print(prompt + ": ")
	if t.scanner.Scan() {
		return t.scanner.Text()
	}
	return ""
}

func (t *terminalPrompter) Secret(prompt string) string {
	fmt.Print(prompt + ": ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not hide input, typing will be visible")
		return t.Input(prompt)
	}
	return string(b)
}

func (t *terminalPrompter) Select(prompt string, options []string) int {
	fmt.Println(prompt)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	for {
		fmt.Print("Select: ")
		if !t.scanner.Scan() {
			return -1
		}
		n, err := strconv.Atoi(strings.TrimSpace(t.scanner.Text()))
		if err != nil || n < 1 || n > len(options) {
			fmt.Printf("Please enter a number between 1 and %d.\n", len(options))
			continue
		}
		return n - 1
	}
}

func (t *terminalPrompter) Print(message string) {
	fmt.Println(message)
}
