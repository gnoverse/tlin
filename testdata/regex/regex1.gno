package main

import (
	"regexp"

	"gno.land/p/demo/ufmt"
)


func avoidRepeatedRegexCompilation(inputs []string) {
    regex := regexp.MustCompile(`\d+`)
    for _, input := range inputs {
        matches := regex.FindAllString(input, -1)
        ufmt.Println(matches)
    }
}
