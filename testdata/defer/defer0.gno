package main

func main() {
	var f func()
	for i := 0; i < 10; i++ {
		defer panic("oops")
	}
	defer f()
}
