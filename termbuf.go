package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// termbuf manages terminal alternate screen buffer
type termbuf struct {
	isActive     bool
	isTerminal   bool
	originalTerm *term.State
	file         *os.File
}

// newTermBuffer creates a new terminal buffer manager
func newTermbuf(w io.Writer) *termbuf {
	// Check if we're writing to a terminal
	f, ok := w.(*os.File)
	isTerminal := ok && term.IsTerminal(int(f.Fd()))

	return &termbuf{
		isActive:   false,
		isTerminal: isTerminal,
		file:       f,
	}
}

// enterAltScreen switches to the alternate screen buffer
func (tb *termbuf) enterAltScreen() error {
	if !tb.isTerminal || tb.isActive {
		return nil
	}

	// Get current terminal settings
	var err error
	tb.originalTerm, err = term.MakeRaw(int(tb.file.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}

	// Save current terminal size for proper formatting
	width, height, err := term.GetSize(int(tb.file.Fd()))
	if err == nil {
		// Set environment variables for terminal dimensions
		// This helps glamour render with the correct width
		os.Setenv("COLUMNS", fmt.Sprintf("%d", width))
		os.Setenv("LINES", fmt.Sprintf("%d", height))
	}

	// Enter alternate screen buffer (smcup)
	if _, err := fmt.Fprint(tb.file, "\033[?1049h"); err != nil {
		return fmt.Errorf("failed to enter alternate screen: %w", err)
	}

	// Clear screen and move cursor to home position
	if _, err := fmt.Fprint(tb.file, "\033[2J\033[H"); err != nil {
		return fmt.Errorf("failed to clear screen: %w", err)
	}

	// Set proper line wrapping mode
	// Enable line wrapping (DECAWM)
	if _, err := fmt.Fprint(tb.file, "\033[?7h"); err != nil {
		return fmt.Errorf("failed to set line wrapping: %w", err)
	}

	// Hide cursor (civis)
	if _, err := fmt.Fprint(tb.file, "\033[?25l"); err != nil {
		return fmt.Errorf("failed to hide cursor: %w", err)
	}

	tb.isActive = true
	return nil
}

// clear clears the screen and resets cursor position with proper spacing
func (tb *termbuf) clear() {
	if tb.isTerminal && tb.isActive {
		fmt.Fprint(tb.file, "\033[2J\033[H")
	}
}

// exitAltScreen returns to the normal screen buffer
func (tb *termbuf) exitAltScreen() error {
	if !tb.isTerminal || !tb.isActive {
		return nil
	}

	// Show cursor (cnorm)
	if _, err := fmt.Fprint(tb.file, "\033[?25h"); err != nil {
		return fmt.Errorf("failed to show cursor: %w", err)
	}

	// Leave alternate screen (rmcup)
	if _, err := fmt.Fprint(tb.file, "\033[?1049l"); err != nil {
		return fmt.Errorf("failed to exit alternate screen: %w", err)
	}

	// Restore terminal state
	if err := term.Restore(int(tb.file.Fd()), tb.originalTerm); err != nil {
		return fmt.Errorf("failed to restore terminal state: %w", err)
	}

	tb.isActive = false
	return nil
}

// normalizeLineEndings ensures consistent line endings and proper spacing
// This helps with the alternate buffer display
func normalizeLineEndings(text string) string {
	// First, normalize all line endings to \n
	text = strings.ReplaceAll(text, "\r\n", "\n")

	// Remove any instances of multiple blank lines that can cause spacing issues
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return text
}

// writeToAlt writes content to the alternate screen with proper spacing
func (tb *termbuf) writeToAlt(content string) error {
	if !tb.isTerminal || !tb.isActive {
		return nil
	}

	// Ensure content has proper line endings for the terminal
	content = strings.ReplaceAll(content, "\n", "\r\n")

	_, err := fmt.Fprint(tb.file, content)
	return err
}

// finalOutput exits the alternate screen and writes the final content to the normal screen
func (tb *termbuf) finalOutput(content string) error {
	// If we're in a terminal and using alt screen
	if tb.isTerminal && tb.isActive {
		if err := tb.exitAltScreen(); err != nil {
			return err
		}

		// Ensure proper line endings for the normal terminal buffer
		content = strings.ReplaceAll(content, "\n", "\r\n")

		// Write the final content to the normal screen
		if _, err := fmt.Fprint(tb.file, content); err != nil {
			return err
		}
		return nil
	}

	// For non-terminal output, just write directly
	_, err := fmt.Fprint(tb.file, content)
	return err
}
