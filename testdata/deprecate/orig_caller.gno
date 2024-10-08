package main

import (
	"std"
)

func main() {
	origCaller := std.GetOrigCaller()
	println(origCaller.String())

	prev := std.PrevRealm()
	prevAddr := prev.Addr()

	println(prevAddr.String())
}
