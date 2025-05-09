package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// SpinnerType represents different styles of spinner animations
type SpinnerType string

const (
	SpinnerDots          SpinnerType = "dots"
	SpinnerDots2         SpinnerType = "dots2"
	SpinnerDots3         SpinnerType = "dots3"
	SpinnerDots4         SpinnerType = "dots4"
	SpinnerDots5         SpinnerType = "dots5"
	SpinnerDots6         SpinnerType = "dots6"
	SpinnerDots7         SpinnerType = "dots7"
	SpinnerDots8         SpinnerType = "dots8"
	SpinnerDots9         SpinnerType = "dots9"
	SpinnerDots10        SpinnerType = "dots10"
	SpinnerDots11        SpinnerType = "dots11"
	SpinnerDots12        SpinnerType = "dots12"
	SpinnerDots13        SpinnerType = "dots13"
	SpinnerLine          SpinnerType = "line"
	SpinnerLine2         SpinnerType = "line2"
	SpinnerPipe          SpinnerType = "pipe"
	SpinnerSimpleDots    SpinnerType = "simpleDots"
	SpinnerStar          SpinnerType = "star"
	SpinnerStar2         SpinnerType = "star2"
	SpinnerFlip          SpinnerType = "flip"
	SpinnerBalloon       SpinnerType = "balloon"
	SpinnerBalloon2      SpinnerType = "balloon2"
	SpinnerNoise         SpinnerType = "noise"
	SpinnerBounce        SpinnerType = "bounce"
	SpinnerBoxBounce     SpinnerType = "boxBounce"
	SpinnerCircle        SpinnerType = "circle"
	SpinnerSquareCorners SpinnerType = "squareCorners"
	SpinnerCircleHalves  SpinnerType = "circleHalves"
	SpinnerToggle        SpinnerType = "toggle"
	SpinnerArrow         SpinnerType = "arrow"
	SpinnerBouncingBar   SpinnerType = "bouncingBar"
	SpinnerBouncingBall  SpinnerType = "bouncingBall"
	SpinnerBinary        SpinnerType = "binary"
)

// spinnerDefinition defines the appearance and behavior of a spinner
type spinnerDefinition struct {
	Interval time.Duration
	Frames   []string
}

// Available spinner definitions
var spinnerDefinitions = map[SpinnerType]spinnerDefinition{
	SpinnerDots: {
		Interval: 60 * time.Millisecond,
		Frames: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
	},
	SpinnerDots2: {
		Interval: 60 * time.Millisecond,
		Frames: []string{
			"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷",
		},
	},
	SpinnerDots3: {
		Interval: 60 * time.Millisecond,
		Frames: []string{
			"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓",
		},
	},
	SpinnerDots4: {
		Interval: 60 * time.Millisecond,
		Frames: []string{
			"⠄", "⠆", "⠇", "⠋", "⠙", "⠸", "⠰", "⠠", "⠰", "⠸", "⠙", "⠋", "⠇", "⠆",
		},
	},
	SpinnerLine: {
		Interval: 115 * time.Millisecond,
		Frames: []string{
			"-", "\\", "|", "/",
		},
	},
	SpinnerLine2: {
		Interval: 75 * time.Millisecond,
		Frames: []string{
			"⠂", "-", "–", "—", "–", "-",
		},
	},
	SpinnerPipe: {
		Interval: 75 * time.Millisecond,
		Frames: []string{
			"┤", "┘", "┴", "└", "├", "┌", "┬", "┐",
		},
	},
	SpinnerSimpleDots: {
		Interval: 200 * time.Millisecond,
		Frames: []string{
			".  ", ".. ", "...", "   ",
		},
	},
	SpinnerStar: {
		Interval: 80 * time.Millisecond,
		Frames: []string{
			"✶", "✸", "✹", "✺", "✹", "✷",
		},
	},
	SpinnerBounce: {
		Interval: 80 * time.Millisecond,
		Frames: []string{
			"⠁", "⠂", "⠄", "⠂",
		},
	},
	SpinnerBoxBounce: {
		Interval: 80 * time.Millisecond,
		Frames: []string{
			"▖", "▘", "▝", "▗",
		},
	},
	SpinnerCircle: {
		Interval: 60 * time.Millisecond,
		Frames: []string{
			"◡", "⊙", "◠",
		},
	},
	SpinnerCircleHalves: {
		Interval: 60 * time.Millisecond,
		Frames: []string{
			"◐", "◓", "◑", "◒",
		},
	},
	SpinnerToggle: {
		Interval: 175 * time.Millisecond,
		Frames: []string{
			"⊶", "⊷",
		},
	},
	SpinnerArrow: {
		Interval: 100 * time.Millisecond,
		Frames: []string{
			"←", "↖", "↑", "↗", "→", "↘", "↓", "↙",
		},
	},
	SpinnerBouncingBar: {
		Interval: 60 * time.Millisecond,
		Frames: []string{
			"[    ]", "[=   ]", "[==  ]", "[=== ]", "[====]", "[ ===]", "[  ==]", "[   =]",
			"[    ]", "[   =]", "[  ==]", "[ ===]", "[====]", "[=== ]", "[==  ]", "[=   ]",
		},
	},
	SpinnerBouncingBall: {
		Interval: 60 * time.Millisecond,
		Frames: []string{
			" ●    ", "  ●   ", "   ●  ", "    ● ", "     ●", "    ● ",
			"   ●  ", "  ●   ", " ●    ", "●     ",
		},
	},
	SpinnerBinary: {
		Interval: 60 * time.Millisecond,
		Frames: []string{
			"010010", "001100", "100101", "111010", "111101", "010111",
			"101011", "111000", "110011", "110101",
		},
	},
}

// Spinner color options
var (
	// Default vibrant color for the spinner
	spinnerColor = lipgloss.Color("#FFFFFF") // Bright green

	// Style for the spinner
	spinnerStyle = lipgloss.NewStyle().
			Bold(true). // Make it bold
			Foreground(spinnerColor)
)

// Spinner manages the animation state for spinner indicators
type Spinner struct {
	definition spinnerDefinition
	current    int
	active     bool
	lastUpdate time.Time
	msgChan    chan struct{}
	stopChan   chan struct{}
	styled     bool // Whether to apply color styling
}

// NewSpinner creates a new spinner with the specified type
func NewSpinner(st SpinnerType) *Spinner {
	def, ok := spinnerDefinitions[st]
	if !ok {
		// Default to dots if the specified spinner is not found
		def = spinnerDefinitions[SpinnerDots]
	}

	return &Spinner{
		definition: def,
		msgChan:    make(chan struct{}, 1),
		stopChan:   make(chan struct{}),
		lastUpdate: time.Now(),
		styled:     true, // Enable styling by default
	}
}

// Start begins the spinner animation in a separate goroutine
func (s *Spinner) Start(w io.Writer) {
	s.active = true

	go func() {
		ticker := time.NewTicker(s.definition.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopChan:
				// Clear the spinner animation
				fmt.Fprint(w, "\r\033[K")
				return

			case <-s.msgChan:
				// Message received, reset animation timer
				s.lastUpdate = time.Now()

			case <-ticker.C:
				// Only show spinner if we've been waiting for a while (100ms)
				if time.Since(s.lastUpdate) > 100*time.Millisecond {
					s.current = (s.current + 1) % len(s.definition.Frames)
					frame := s.definition.Frames[s.current]

					// Apply styling if enabled
					if s.styled {
						frame = spinnerStyle.Render(frame)
					}

					fmt.Fprintf(w, "\r\033[K%s", frame) // Clear line and print frame
				}
			}
		}
	}()
}

// Update signals that new data was received
func (s *Spinner) Update() {
	if s.active {
		// Non-blocking send to avoid hangs if channel is full
		select {
		case s.msgChan <- struct{}{}:
		default:
		}
	}
}

// Stop terminates the spinner animation
func (s *Spinner) Stop() {
	if s.active {
		s.active = false
		close(s.stopChan)
	}
}

// SetColor changes the spinner color
func (s *Spinner) SetColor(color string) {
	spinnerStyle = spinnerStyle.Copy().Foreground(lipgloss.Color(color))
}

// DisableStyling turns off color and bold styling
func (s *Spinner) DisableStyling() {
	s.styled = false
}

// EnableStyling turns on color and bold styling
func (s *Spinner) EnableStyling() {
	s.styled = true
}

// GetSpinnerType returns the appropriate spinner type based on user preference
func GetSpinnerType(spinnerStyle string) SpinnerType {
	switch spinnerStyle {
	case "dots":
		return SpinnerDots
	case "dots2":
		return SpinnerDots2
	case "dots3":
		return SpinnerDots3
	case "dots4":
		return SpinnerDots4
	case "line":
		return SpinnerLine
	case "simpleDots":
		return SpinnerSimpleDots
	case "star":
		return SpinnerStar
	case "bounce":
		return SpinnerBounce
	case "boxBounce":
		return SpinnerBoxBounce
	case "circle":
		return SpinnerCircle
	case "arrow":
		return SpinnerArrow
	case "binary":
		return SpinnerBinary
	case "bouncingBar":
		return SpinnerBouncingBar
	case "bouncingBall":
		return SpinnerBouncingBall
	default:
		return SpinnerDots // Default to dots
	}
}

// demonstrateSpinner shows a live animation of a specific spinner type
func demonstrateSpinner(spinnerName string, colorStr string) error {
	spinnerType := GetSpinnerType(spinnerName)

	// If spinner type doesn't exist, show error
	if _, ok := spinnerDefinitions[spinnerType]; !ok {
		fmt.Printf("Unknown spinner type: %s\n", spinnerName)
		fmt.Println("Run 'glow spinner' without arguments to see available spinner types")
		return nil
	}

	fmt.Printf("Demonstrating '%s' spinner...", spinnerName)
	if spinnerFlags.duration > 0 {
		fmt.Printf(" (Will run for %s)\n\n", spinnerFlags.duration)
	} else if spinnerFlags.autoQuit {
		fmt.Printf(" (Will show all frames once)\n\n")
	} else {
		fmt.Printf(" (Press Ctrl+C to exit)\n\n")
	}

	// Create and configure the spinner
	sp := NewSpinner(spinnerType)

	// Apply custom color if specified
	if colorStr != "" {
		sp.SetColor(colorStr)
	}

	// Get the spinner definition
	definition, _ := spinnerDefinitions[spinnerType]

	// Start the spinner
	sp.Start(os.Stdout)

	// Wait for user interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Show info while spinner is running
	go func() {
		time.Sleep(500 * time.Millisecond)
		fmt.Print("\r\033[K")

		fmt.Printf("\rSpinner: %s   Frames: %d   Interval: %s   Color: %s\n\n",
			spinnerName,
			len(definition.Frames),
			definition.Interval.String(),
			colorStr)

		fmt.Println("To use this spinner in Glow:")
		fmt.Printf("  glow --spinner=%s --spinner-color=%s -\n\n",
			spinnerName, colorStr)
	}()

	// Set up timeout based on flags
	var timeout <-chan time.Time
	if spinnerFlags.duration > 0 {
		timeout = time.After(spinnerFlags.duration)
	} else if spinnerFlags.autoQuit {
		// Show each frame exactly once (one full animation cycle)
		cycleTime := definition.Interval * time.Duration(len(definition.Frames))
		// Add a little buffer to ensure we see the complete cycle
		timeout = time.After(cycleTime + 100*time.Millisecond)
	}

	// Wait for Ctrl+C or timeout
	select {
	case <-quit:
		// User interrupted
	case <-timeout:
		// Duration elapsed or showed all frames once
	}

	// Clean up the spinner
	sp.Stop()
	fmt.Println("\nSpinner demonstration ended.")

	return nil
}

// showSpinnerGallery displays all available spinner animations
func showSpinnerGallery() error {
	fmt.Println("Available spinner animations for Glow")
	fmt.Println("Use with --spinner=NAME")
	fmt.Println("To see a live demo of a specific spinner, run: glow spinner NAME")
	fmt.Println()

	// Get terminal dimensions for better display
	width := 60
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			width = w
		}
	}

	// Map of spinner types to preview
	spinners := []struct {
		name  string
		stype SpinnerType
	}{
		{"dots", SpinnerDots},
		{"dots2", SpinnerDots2},
		{"dots3", SpinnerDots3},
		{"dots4", SpinnerDots4},
		{"line", SpinnerLine},
		{"line2", SpinnerLine2},
		{"pipe", SpinnerPipe},
		{"simpleDots", SpinnerSimpleDots},
		{"star", SpinnerStar},
		{"star2", SpinnerStar2},
		{"flip", SpinnerFlip},
		{"balloon", SpinnerBalloon},
		{"balloon2", SpinnerBalloon2},
		{"bounce", SpinnerBounce},
		{"boxBounce", SpinnerBoxBounce},
		{"circle", SpinnerCircle},
		{"squareCorners", SpinnerSquareCorners},
		{"circleHalves", SpinnerCircleHalves},
		{"toggle", SpinnerToggle},
		{"arrow", SpinnerArrow},
		{"bouncingBar", SpinnerBouncingBar},
		{"bouncingBall", SpinnerBouncingBall},
		{"binary", SpinnerBinary},
	}

	// Calculate columns for display
	cols := 3
	if width < 80 {
		cols = 2
	}
	if width < 40 {
		cols = 1
	}

	// Style for spinner name
	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		Bold(false)

	// Style for spinner separators
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555"))

	// Display each spinner with its name and a preview
	for i, s := range spinners {
		def, ok := spinnerDefinitions[s.stype]
		if !ok {
			continue
		}

		// Preview first 4 frames of each spinner
		previewFrames := def.Frames
		if len(previewFrames) > 4 {
			previewFrames = previewFrames[:4]
		}

		// Apply styling to each frame
		styledFrames := make([]string, len(previewFrames))
		for j, frame := range previewFrames {
			styledFrames[j] = spinnerStyle.Render(frame)
		}

		preview := strings.Join(styledFrames, sepStyle.Render(" "))
		nameWidth := 15

		// Format output based on columns
		if cols == 1 || i%cols == 0 {
			fmt.Printf("%s %s %s\n",
				nameStyle.Render(fmt.Sprintf("%-*s", nameWidth, s.name)),
				sepStyle.Render(":"),
				preview)
		} else {
			fmt.Printf("%s %s %-20s",
				nameStyle.Render(fmt.Sprintf("%-*s", nameWidth, s.name)),
				sepStyle.Render(":"),
				preview)
			if (i+1)%cols == 0 {
				fmt.Println()
			}
		}
	}

	fmt.Println("\nExample usage:")
	fmt.Println("  glow --spinner=dots3 -")
	fmt.Println("  cat README.md | glow --spinner=bouncingBall -")
	fmt.Println("\nTo see a specific spinner in action:")
	fmt.Println("  glow spinner dots3")

	return nil
}

// demonstrateAllSpinners shows each spinner animation in sequence
func demonstrateAllSpinners(colorStr string) error {
	fmt.Println("Demonstrating all spinner animations...")
	fmt.Println("Each spinner will run for 3 seconds")
	fmt.Println("Press Ctrl+C at any time to exit")
	fmt.Println()

	// Create a list of spinner types to demonstrate
	spinners := []struct {
		name  string
		stype SpinnerType
	}{
		{"dots", SpinnerDots},
		{"dots2", SpinnerDots2},
		{"dots3", SpinnerDots3},
		{"dots4", SpinnerDots4},
		{"line", SpinnerLine},
		{"line2", SpinnerLine2},
		{"pipe", SpinnerPipe},
		{"simpleDots", SpinnerSimpleDots},
		{"star", SpinnerStar},
		{"star2", SpinnerStar2},
		{"flip", SpinnerFlip},
		{"balloon", SpinnerBalloon},
		{"balloon2", SpinnerBalloon2},
		{"bounce", SpinnerBounce},
		{"boxBounce", SpinnerBoxBounce},
		{"circle", SpinnerCircle},
		{"squareCorners", SpinnerSquareCorners},
		{"circleHalves", SpinnerCircleHalves},
		{"toggle", SpinnerToggle},
		{"arrow", SpinnerArrow},
		{"bouncingBar", SpinnerBouncingBar},
		{"bouncingBall", SpinnerBouncingBall},
		{"binary", SpinnerBinary},
	}

	// Set up signal handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Demonstrate each spinner
	for i, s := range spinners {
		// Check if user interrupted
		select {
		case <-quit:
			fmt.Println("\nDemonstration interrupted.")
			return nil
		default:
			// Continue
		}

		// Create the spinner
		sp := NewSpinner(s.stype)
		if colorStr != "" {
			sp.SetColor(colorStr)
		}

		// Show spinner info
		fmt.Printf("\r\033[K%d/%d: '%s' spinner\n", i+1, len(spinners), s.name)

		// Start the spinner
		sp.Start(os.Stdout)

		// Display for 3 seconds or until user interrupts
		select {
		case <-time.After(3 * time.Second):
			// Time's up for this spinner
		case <-quit:
			sp.Stop()
			fmt.Println("\nDemonstration interrupted.")
			return nil
		}

		// Stop the spinner
		sp.Stop()
		fmt.Println()
	}

	fmt.Println("All spinners demonstrated!")
	return nil
}
