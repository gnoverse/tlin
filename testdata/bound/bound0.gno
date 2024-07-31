// ref: https://github.com/golangci/golangci-lint/discussions/4150

package main

type Item struct {
    Name  string
    Value int
}

func main() {
    sourceItems := []*Item{
        {"item1", 10},
        {"item2", 20},
        {"item3", 30},
    }

    destinationItems := make([]*Item, 0, len(sourceItems))

    i := 0
    for _, item := range sourceItems {
        destinationItems[i] = item // Expect out-of-range linter warning here
        i++
    }
}
