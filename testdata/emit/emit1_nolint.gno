package main

import "runtime/chain"

func main() {
	//nolint:emit-format
    chain.Emit("OwnershipChange", "newOwner", newOwner.String(), "oldOwner", oldOwner.String())
}
