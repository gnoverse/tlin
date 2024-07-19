package main

import "strings"

func foo(x, y bool) int {
	if x {
		return 1
	} else {
		return 2
	}
}

func bar() {
	println("hello")
}

func main() {
	x := foo(true)
	println(x)
}
