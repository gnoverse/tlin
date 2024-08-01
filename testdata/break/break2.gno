package main

func main() {
outer:
	for {
		switch x := 1; x {
		case 1:
			println("one")
			break outer
		case 2:
			println("two")
		}
	}
}
