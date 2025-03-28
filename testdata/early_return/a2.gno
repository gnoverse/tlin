package main

func example(x int) string {
    if x > 10 {
        doSomething()
        return "greater"
    } else {
        doSomethingElse()
        return "less or equal"
    }
}
