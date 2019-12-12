package main

import "fmt"

type Logger struct {
	Input chan string
	Quit  chan bool
}

func NewLogger() Logger {
	l := Logger{
		Input: make(chan string),
		Quit:  make(chan bool),
	}
	go func() {

	Exit:
		for {
			select {
			case msg := <-l.Input:
				if !rootQuiet {
					fmt.Printf("%s\n\n", msg)
				}
			case <-l.Quit:
				break Exit
			}
		}
	}()
	return l
}
