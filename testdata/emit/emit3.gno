package main

import "std"

func main() {

	anotherOwner := anotherOwner.String()
	std.Emit(
		"OwnershipChange",
		"newOwner", newOwner.String(),
		"oldOwner", oldOwner.String(),
		"anotherOwner",
		anotherOwner,
	)
}
