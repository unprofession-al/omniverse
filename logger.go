package main

import "fmt"

// Logger manages printing to stdout in a central location
type Logger struct {
	input chan string
	quit  chan bool
	quiet bool
}

// NewLogger returns a logger instance
func NewLogger(quiet bool) Logger {
	l := Logger{
		input: make(chan string),
		quit:  make(chan bool),
		quiet: quiet,
	}

	go func() {
		l.run()
	}()

	return l
}

func (l Logger) run() {
	for {
		select {
		case msg := <-l.input:
			if !l.quiet {
				fmt.Printf("%s\n\n", msg)
			}
		case <-l.quit:
			return
		}
	}
}

// Write implements io.Writer and prints to stdout
func (l Logger) Write(p []byte) (n int, err error) {
	l.input <- string(p)
	return len(p), nil
}

// Quit ends the control loop and terminates the Logger
func (l Logger) Quit() {
	l.quit <- true
}
