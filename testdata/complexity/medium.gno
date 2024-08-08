package testdata

func mediumComplexity(x, y int) int {
    if x > y {
        if x > 10 {
            return x * 2
        } else if x > 5 {
            return x + y
        } else {
            return x - y
        }
    } else if y > 10 {
        return y * 2
    } else {
        return x + y
    }
}

func anotherMediumComplexity(s string) string {
    result := ""
    for _, c := range s {
        if c >= 'a' && c <= 'z' {
            result += string(c - 32)
        } else if c >= 'A' && c <= 'Z' {
            result += string(c + 32)
        } else {
            result += string(c)
        }
    }
    return result
}