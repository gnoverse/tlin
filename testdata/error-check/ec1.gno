package main

import "errors"

func foo() (int, error) {
	return 0, errors.New("test")
}

func main() {
	v, err := foo()
	if err != nil {
		panic(err)
	}
	println(v)
}
