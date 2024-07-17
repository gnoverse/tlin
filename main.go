package main

// import "fmt"

func foo(x bool) int {
	if x {
		return 1
	} else {
		return 2
	}
}

// func bar() {
// 	x := 5
// 	fmt.Printf("hello")
// }

func main() {
	x := foo(true)
	println(x)
}
