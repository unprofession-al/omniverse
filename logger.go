package main

import "fmt"

type Logger struct {
	input chan string
	quit  chan bool
	quiet bool
}

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

func (l Logger) Write(p []byte) (n int, err error) {
	l.input <- string(p)
	return len(p), nil
}

func (l Logger) Quit() {
	l.quit <- true
}
