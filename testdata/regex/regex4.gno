package main

import "regexp"

func greedyQuantifierExample(largeInput string) {
    regex := regexp.MustCompile(`<.*>`)
    matches := regex.FindAllString(largeInput, -1)
}

// minimized:
// <.*?>
