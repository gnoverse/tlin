package main

import "gno.land/p/nt/ufmt"

func main() {
	_ = ufmt.Sprintf("plain literal without any verb")
	ufmt.Printf("another literal")
}
