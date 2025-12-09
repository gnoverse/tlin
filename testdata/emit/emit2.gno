package main

import "runtime/chain"

func main() {
    chain.Emit(
		"OwnershipChange",
		"newOwner", newOwner.String(),
		"oldOwner", oldOwner.String(),
	)
}
