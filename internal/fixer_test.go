package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImproveCode(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "don't need to modify",
			input: `package main

func foo(x bool) int {
	if x {
		println("x")
	} else {
		println("hello")
	}
}`,
			expected: `package main

func foo(x bool) int {
	if x {
		println("x")
	} else {
		println("hello")
	}
}`,
		},
		{
			name: "Remove unnecessary else",
			input: `
package main

func unnecessaryElse() bool {
    if condition {
        return true
    } else {
        return false
    }
}`,
			expected: `package main

func unnecessaryElse() bool {
	if condition {
		return true
	}
	return false

}`,
		},
		{
			name: "Keep necessary else",
			input: `
package main

func necessaryElse() int {
    if condition {
        return 1
    } else {
        doSomething()
        return 2
    }
}`,
			expected: `package main

func necessaryElse() int {
	if condition {
		return 1
	}
	doSomething()
	return 2

}`,
		},
		//         {
		//             name: "Multiple unnecessary else",
		//             input: `
		// package main

		// func multipleUnnecessaryElse() int {
		//     if condition1 {
		//         return 1
		//     } else {
		//         if condition2 {
		//             return 2
		//         } else {
		//             return 3
		//         }
		//     }
		// }`,
		//             expected: `package main

		// func multipleUnnecessaryElse() int {
		//     if condition1 {
		//         return 1
		//     }
		//     if condition2 {
		//         return 2
		//     }
		//     return 3
		// }
		// `,
		//         },
		//         {
		//             name: "Mixed necessary and unnecessary else",
		//             input: `
		// package main

		// func mixedElse() int {
		//     if condition1 {
		//         return 1
		//     } else {
		//         if condition2 {
		//             doSomething()
		//             return 2
		//         } else {
		//             return 3
		//         }
		//     }
		// }`,
		//             expected: `package main

		// func mixedElse() int {
		//     if condition1 {
		//         return 1
		//     } else {
		//         if condition2 {
		//             doSomething()
		//             return 2
		//         }
		//         return 3
		//     }
		// }
		// `,
		//         },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			improved, err := improveCode([]byte(tc.input))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, improved, "Improved code does not match expected output")
		})
	}
}
