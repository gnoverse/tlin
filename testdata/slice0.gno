package slice

func foo() {
	slice := []int{1, 2, 3}
	_ = slice[1:len(slice)]
}

func bar() {
	slice := []int{1, 2, 3}
	_ = slice[:len(slice)]
}
