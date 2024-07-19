package slice

var slice = []int{1, 2, 3}

func foo() {
	_ = slice[1:len(slice)]
}

func bar() {
	_ = slice[:len(slice)]
}

func baz() {
	a := 1
	_ = slice[a:len(slice)]
}
