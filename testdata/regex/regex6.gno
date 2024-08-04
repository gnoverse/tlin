package main

import "regexp"

func withRepeat() {
	r1 := regexp.MustCompile("pattern")
	r2 := regexp.MustCompile("pattern")
	_ = r1
	_ = r2
}
