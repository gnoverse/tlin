package registry

type Tree struct{ size int }

var store = &Tree{}

func GetStore() *Tree { return store }
