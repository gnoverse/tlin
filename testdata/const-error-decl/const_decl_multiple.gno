package main

import "errors"

const (
	err = errors.New("error")
	err2 = errors.New("error2")
)

func main() {
	println("foo")
}
