package main

import (
	"regexp"

	"gno.land/p/demo/ufmt"
)

func repeatedRegexCompilation(inputs []string) {
    for _, input := range inputs {
        regex := regexp.MustCompile(`\d+`)
        matches := regex.FindAllString(input, -1)
        ufmt.Println(matches)
    }
}
