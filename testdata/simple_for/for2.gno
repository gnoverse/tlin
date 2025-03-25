package main

func Range() int {
	return 5
}

func main() {
	for i := 0; i < Range(); i++ {
		println(i)
	}
}
