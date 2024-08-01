package main

func main() {
	switch x := 1; x {
	case 1:
		println("one")
		break
	case 2:
		println("two")
	default:
		println("other")
		break
	}
}
