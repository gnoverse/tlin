package main

import "gno.land/p/nt/ufmt"

func main() {
	_ = ufmt.Sprintf("hello %s", "world")
	ufmt.Printf("count=%d\n", 42)
}
