package main

import "runtime/chain"

func main() {

	anotherOwner := anotherOwner.String()
	chain.Emit(
		"OwnershipChange",
		"newOwner", newOwner.String(),
		"oldOwner", oldOwner.String(),
		"anotherOwner",
		anotherOwner,
	)
}
