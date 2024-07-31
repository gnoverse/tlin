package main

import "std"

func main() {
    std.Emit(
		"OwnershipChange",
		"newOwner", newOwner.String(),
		"oldOwner", 
		oldOwner.String(),
		"anotherOwner", anotherOwner.String(),
	)
}
