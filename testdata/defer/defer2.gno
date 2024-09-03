package main

func foo() (result int) {
	defer func() {
		result = 42
		return
	}()
	return 0
}
