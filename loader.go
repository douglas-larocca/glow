package main

import (
	"fmt"
	"io"
	"time"
)

// loaderType represents different styles of loading animations
type loaderType int

const (
	loaderDots loaderType = iota
	loaderBraille
)

// loader manages the animation state for loading indicators
type loader struct {
	loaderType loaderType
	frames     []string
	current    int
	active     bool
	lastUpdate time.Time
	msgChan    chan struct{}
	stopChan   chan struct{}
}

// newLoader creates a new loader with the specified type
func newLoader(lt loaderType) *loader {
	var frames []string

	switch lt {
	case loaderDots:
		frames = []string{".", "..", "...", ""}
	case loaderBraille:
		frames = []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		}
	}

	return &loader{
		loaderType: lt,
		frames:     frames,
		msgChan:    make(chan struct{}, 1),
		stopChan:   make(chan struct{}),
		lastUpdate: time.Now(),
	}
}

// start begins the loader animation in a separate goroutine
func (l *loader) start(w io.Writer) {
	l.active = true

	go func() {
		ticker := time.NewTicker(40 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-l.stopChan:
				// Clear the loader animation
				fmt.Fprint(w, "\r\033[K")
				return

			case <-l.msgChan:
				// Message received, reset animation timer
				l.lastUpdate = time.Now()

			case <-ticker.C:
				// Only show loader if we've been waiting for a while (500ms)
				if time.Since(l.lastUpdate) > 20*time.Millisecond {
					l.current = (l.current + 1) % len(l.frames)
					frame := l.frames[l.current]
					fmt.Fprintf(w, "\r\033[K%s", frame) // Clear line and print frame
				}
			}
		}
	}()
}

// update signals that new data was received
func (l *loader) update() {
	if l.active {
		// Non-blocking send to avoid hangs if channel is full
		select {
		case l.msgChan <- struct{}{}:
		default:
		}
	}
}

// stop terminates the loader animation
func (l *loader) stop() {
	if l.active {
		l.active = false
		close(l.stopChan)
	}
}
