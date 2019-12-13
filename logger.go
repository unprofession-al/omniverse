package main

import "fmt"

type logger struct {
	Input chan string
	Quit  chan bool
}

func NewLogger() logger {
	l := logger{
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
