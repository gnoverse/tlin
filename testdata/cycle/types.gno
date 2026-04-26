package main

type A struct {
	b *B
}

type B struct {
	a *A
}
