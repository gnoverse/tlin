package main

func main() {
	for i := 0; i < 10; i++ {
		_ = make([]int, 10)
		_ = new(string)
	}
}
