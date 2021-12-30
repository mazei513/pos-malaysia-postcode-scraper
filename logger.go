package main

import "fmt"

type logger struct{ quiet bool }

func (l logger) println(a ...interface{}) {
	if l.quiet {
		return
	}
	fmt.Println(a...)
}
