package main

import "std"

func main() {
	//nolint:emit-format
    std.Emit("OwnershipChange", "newOwner", newOwner.String(), "oldOwner", oldOwner.String())
}
