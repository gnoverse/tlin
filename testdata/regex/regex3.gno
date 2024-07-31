package main

import "regexp"

func unnecessaryCaptureGroupExample() {
    regex := regexp.MustCompile(`(https?://\S+)`)
}

// minimized:
// (?:https?://\S+)`
