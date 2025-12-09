package main

import (
    "fmt"
    "runtime/chain"
    "strings" // unused import
    _ "gno.land/p/demo/diff" // unused import but ignored

    "gno.land/p/demo/ufmt"
)

func main() {
    fmt.Println("Hello, World!")
    ufmt.Printf("Formatted: %s\n", "test")

    chain.Emit(
        "foo",
        "1", "2",
    )
}
