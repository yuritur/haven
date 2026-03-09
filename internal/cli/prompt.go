package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

type terminalPrompter struct{}

func (t *terminalPrompter) Confirm(message string) bool {
	fmt.Print(message + " [Y/n] ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		switch strings.ToLower(input) {
		case "", "y", "yes":
			return true
		}
	}
	return false
}

func (t *terminalPrompter) Input(prompt string) string {
	fmt.Print(prompt + ": ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text()
	}
	return ""
}

func (t *terminalPrompter) Secret(prompt string) string {
	fmt.Print(prompt + ": ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after hidden input
	if err != nil {
		return t.Input(prompt)
	}
	return string(b)
}

func (t *terminalPrompter) Select(prompt string, options []string) int {
	fmt.Println(prompt)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Select: ")
		if !scanner.Scan() {
			return -1
		}
		n, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
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
