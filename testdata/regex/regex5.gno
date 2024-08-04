package main

import (
	"regexp"
)

func preferStringMethodExample(input string) bool {
    return regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(input)
}

// Suggestion:
// for _, char := range input {
//     if !unicode.IsLetter(char) {
//         return false
//     }
// }
// return true
