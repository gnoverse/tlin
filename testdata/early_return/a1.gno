package main

func example(x, y int) string {
    if x > 10 {
        if y > 20 {
            return "x > 10, y > 20"
        } else {
            return "x > 10, y <= 20"
        }
    } else {
        return "x <= 10"
    }
}
