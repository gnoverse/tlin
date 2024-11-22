package main

import "errors"

func foo() (int, error) {
	return 0, errors.New("test")
}

func main() {
	v, _ := foo()
	println(v)
}
