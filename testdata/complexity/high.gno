package testdata

func highComplexity(x, y, z int) int {
    result := 0
    if x > y {
        if x > z {
            result = x
        } else {
            result = z
        }
    } else {
        if y > z {
            result = y
        } else {
            result = z
        }
    }

    for i := 0; i < result; i++ {
        if i%2 == 0 {
            result += i
        } else if i%3 == 0 {
            result += i * 2
        } else if i%5 == 0 {
            result += i * 3
        } else {
            result -= i
        }
    }

    switch {
    case result < 100:
        return result * 2
    case result < 1000:
        return result * 3
    case result < 10000:
        return result * 4
    default:
        return result
    }
}

func anotherHighComplexity(s string) int {
    count := 0
    for i, c := range s {
        switch {
        case c >= 'a' && c <= 'z':
            if i%2 == 0 {
                count += int(c - 'a' + 1)
            } else {
                count += int(c - 'a' + 2)
            }
        case c >= 'A' && c <= 'Z':
            if i%3 == 0 {
                count += int(c - 'A' + 1)
            } else if i%3 == 1 {
                count += int(c - 'A' + 2)
            } else {
                count += int(c - 'A' + 3)
            }
        case c >= '0' && c <= '9':
            digit := int(c - '0')
            if digit%2 == 0 {
                count += digit * 2
            } else {
                count += digit * 3
            }
        default:
            count++
        }
    }
    return count
}
