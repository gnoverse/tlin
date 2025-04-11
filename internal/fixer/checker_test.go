package fixer

import (
	"go/token"
	"os"
	"testing"

	tt "github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCFGEquivalence(t *testing.T) {
	tests := []struct {
		name                string
		originalCode        string
		modifiedCode        string
		expectEqual         bool
		confidenceThreshold float64
	}{
		{
			name: "Remove unnecessary else - equivalent",
			originalCode: `package main

func checkValue(n int) bool {
	if n > 10 {
		return true
	} else {
		return false
	}
}`,
			modifiedCode: `package main

func checkValue(n int) bool {
	if n > 10 {
		return true
	}
	return false
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			// this case would not happen in practice,
			// but it's good to test AST normalization works
			name: "Remove unnecessary else - with many new lines",
			originalCode: `package main

func checkValue(n int) bool {
	if n > 10 {
		return true
	} else {
		return false
	}
}`,
			modifiedCode: `package main

func checkValue(n int) bool {


	if n > 10 {

		return true


		}


	return false
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Simplify slice expression - equivalent",
			originalCode: `package main

func getSlice() []int {
	slice := []int{1, 2, 3}
	return slice[:len(slice)]
}`,
			modifiedCode: `package main

func getSlice() []int {
	slice := []int{1, 2, 3}
	return slice[:]
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Reformat emit function - equivalent",
			originalCode: `package main

import "std"

func emitEvent() {
	newOwner := "Alice"
	oldOwner := "Bob"
	std.Emit("OwnershipChange", "newOwner", newOwner, "oldOwner", oldOwner)
}`,
			modifiedCode: `package main

import "std"

func emitEvent() {
	newOwner := "Alice"
	oldOwner := "Bob"
	std.Emit(
		"OwnershipChange",
		"newOwner", newOwner,
		"oldOwner", oldOwner,
	)
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Change conditional logic - NOT equivalent",
			originalCode: `package main

func processValue(n int) string {
	if n > 10 {
		return "large"
	} else if n > 5 {
		return "medium"
	} else {
		return "small"
	}
}`,
			modifiedCode: `package main

func processValue(n int) string {
	if n >= 10 {  // Changed from > to >=
		return "large"
	} else if n > 5 {
		return "medium"
	} else {
		return "small"
	}
}`,
			expectEqual:         false,
			confidenceThreshold: 0.8,
		},
		{
			name: "Add conditional branch - NOT equivalent",
			originalCode: `package main

func processValue(n int) string {
	if n > 10 {
		return "large"
	} 
	return "small"
}`,
			modifiedCode: `package main

func processValue(n int) string {
	if n > 10 {
		return "large"
	} else if n > 5 {  // Added branch
		return "medium"
	}
	return "small"
}`,
			expectEqual:         false,
			confidenceThreshold: 0.8,
		},
		{
			name: "Change loop structure - NOT equivalent",
			originalCode: `package main

func sumValues(nums []int) int {
	sum := 0
	for i := 0; i < len(nums); i++ {
		sum += nums[i]
	}
	return sum
}`,
			modifiedCode: `package main

func sumValues(nums []int) int {
	sum := 0
	for _, num := range nums {  // Changed loop structure
		sum += num
	}
	return sum
}`,
			expectEqual:         false,
			confidenceThreshold: 0.8,
		},
		{
			name: "Empty function - equivalent",
			originalCode: `package main

func doNothing() {
}`,
			modifiedCode: `package main

func doNothing() {
	// Just a comment
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Complex nested structure - equivalent",
			originalCode: `package main

func complexFunction(a, b int) int {
	result := 0
	if a > b {
		for i := 0; i < a; i++ {
			if i%2 == 0 {
				result += i
			} else {
				result -= i
			}
		}
	} else {
		for i := 0; i < b; i++ {
			if i%2 == 0 {
				result += i
			} else {
				result -= i
			}
		}
	}
	return result
}`,
			modifiedCode: `package main

func complexFunction(a, b int) int {
	result := 0
	if a > b {
		for i := 0; i < a; i++ {
			if i%2 == 0 {
				result += i
			} else {
				result -= i
			}
		}
	} else {
		// Just reformatted this part
		for i := 0; i < b; i++ {
			if i%2 == 0 {
				result += i
			} else {
				result -= i
			}
		}
	}
	return result
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Multiple return paths - equivalent",
			originalCode: `package main

func process(value int) (int, error) {
	if value < 0 {
		return 0, errors.New("negative value")
	}
	
	result := value * 2
	if result > 100 {
		return 100, nil
	}
	
	return result, nil
}`,
			modifiedCode: `package main

func process(value int) (int, error) {
	// Added comment
	if value < 0 {
		return 0, errors.New("negative value")
	}
	
	result := value * 2  // Changed variable name
	if result > 100 {
		return 100, nil
	}
	
	return result, nil
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Different variable name - equivalent",
			originalCode: `package main

func double(x int) int {
	return x * 2
}`,
			modifiedCode: `package main

func double(y int) int {  // Changed parameter name
	return y * 2
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Type switch statement - equivalent",
			originalCode: `package main

func processValue(x interface{}) string {
	switch v := x.(type) {
	case int:
		return "integer"
	case string:
		return "string"
	default:
		return "unknown"
	}
}`,
			modifiedCode: `package main

func processValue(x interface{}) string {
	switch v := x.(type) {
	case int:
		return "integer"
	case string:
		return "string"
	default:
		return "unknown"
	}
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewContentBasedCFGChecker(tt.confidenceThreshold, true)
			isEqual, report, err := checker.CheckEquivalence(tt.originalCode, tt.modifiedCode)

			require.NoError(t, err, "CFG equivalence check should not produce an error")

			if tt.expectEqual {
				assert.True(t, isEqual, "Expected code to be equivalent, but checker reported non-equivalence:\n%s", report)
			} else {
				assert.False(t, isEqual, "Expected code to be non-equivalent, but checker reported equivalence")
			}
			t.Logf("Equivalence report:\n%s", report)
		})
	}
}

func TestIntegrationWithFixer(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		issues      []tt.Issue
		expectEqual bool
	}{
		{
			name: "Simplify slice range - should maintain CFG equivalence",
			input: `package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:len(slice)]
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 5, Column: 5},
					End:        token.Position{Line: 5, Column: 24},
					Suggestion: "_ = slice[:]",
					Confidence: 0.9,
				},
			},
			expectEqual: true,
		},
		{
			name: "Reformat emit function - should maintain CFG equivalence",
			input: `package main

import "std"

func main() {
	newOwner := "Alice"
	oldOwner := "Bob"
	std.Emit("OwnershipChange",
	"newOwner", newOwner, "oldOwner", oldOwner)
}`,
			issues: []tt.Issue{
				{
					Rule:    "emit-format",
					Message: "Consider formatting std.Emit call for better readability",
					Start:   token.Position{Line: 8, Column: 5},
					End:     token.Position{Line: 9, Column: 44},
					Suggestion: `std.Emit(
    "OwnershipChange",
    "newOwner", newOwner,
    "oldOwner", oldOwner,
)`,
					Confidence: 0.9,
				},
			},
			expectEqual: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, testFile, cleanup := setupTestFile(t, tc.input)
			defer cleanup()

			for i := range tc.issues {
				tc.issues[i].Filename = testFile
			}

			originalCode, err := readFile(t, testFile)
			require.NoError(t, err, "Failed to read original code")

			// apply the auto fix
			fixer := New(false, 0.8)
			err = fixer.Fix(testFile, tc.issues)
			require.NoError(t, err, "Fix application should not produce an error")

			// read the fixed code
			fixedCode, err := readFile(t, testFile)
			require.NoError(t, err, "Failed to read fixed code")

			// check the CFG equivalence
			checker := NewContentBasedCFGChecker(0.8, true)
			isEqual, report, err := checker.CheckEquivalence(originalCode, fixedCode)
			require.NoError(t, err, "CFG equivalence check should not produce an error")

			if tc.expectEqual {
				assert.True(t, isEqual, "Expected CFGs to be equivalent after fix, but got non-equivalence:\n%s", report)
			} else {
				assert.False(t, isEqual, "Expected CFGs to be non-equivalent after fix, but got equivalence")
			}

			t.Logf("Equivalence report for fix:\n%s", report)
		})
	}
}

func readFile(t *testing.T, filename string) (string, error) {
	t.Helper()
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func TestCFGEquivalence2(t *testing.T) {
	tests := []struct {
		name                string
		originalCode        string
		modifiedCode        string
		expectEqual         bool
		confidenceThreshold float64
	}{
		{
			name: "Function with defer - equivalent",
			originalCode: `package main

import "fmt"

func withDefer() {
	defer fmt.Println("cleanup")
	fmt.Println("working")
}`,
			modifiedCode: `package main

import "fmt"

func withDefer() {
	defer fmt.Println("cleanup")
	fmt.Println("working")  // Added comment
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Function with recovered panic - equivalent",
			originalCode: `package main

import "fmt"

func withRecover() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered:", r)
		}
	}()
	panic("something went wrong")
}`,
			modifiedCode: `package main

import "fmt"

func withRecover() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered:", r)
		}
	}()
	panic("something went wrong")  // Added comment
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Switch with fallthrough - equivalent",
			originalCode: `package main

import "fmt"

func withSwitch(n int) {
	switch n {
	case 1:
		fmt.Println("one")
		fallthrough
	case 2:
		fmt.Println("two")
	default:
		fmt.Println("other")
	}
}`,
			modifiedCode: `package main

import "fmt"

func withSwitch(n int) {
	switch n {
	case 1:
		fmt.Println("one")  // Added comment
		fallthrough
	case 2:
		fmt.Println("two")
	default:
		fmt.Println("other")
	}
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Type switch - equivalent",
			originalCode: `package main

import "fmt"

func typeSwitch(val interface{}) {
	switch v := val.(type) {
	case int:
		fmt.Println("int:", v)
	case string:
		fmt.Println("string:", v)
	default:
		fmt.Println("unknown type")
	}
}`,
			modifiedCode: `package main

import "fmt"

func typeSwitch(val interface{}) {
	switch v := val.(type) {
	case int:
		fmt.Println("int:", v)
	case string:
		fmt.Println("string:", v)  // Added comment
	default:
		fmt.Println("unknown type")
	}
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Function with labels and goto - equivalent",
			originalCode: `package main

import "fmt"

func withGoto() {
	i := 0
start:
	i++
	if i < 10 {
		goto start
	}
}`,
			modifiedCode: `package main

import "fmt"

func withGoto() {
	i := 0
start:  // Added comment
	i++
	if i < 10 {
		goto start
	}
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Added code after return - NOT equivalent",
			originalCode: `package main

func earlyReturn(n int) int {
	if n < 0 {
		return 0
	}
	return n * 2
}`,
			modifiedCode: `package main

func earlyReturn(n int) int {
	if n < 0 {
		return 0
		n = 0  // Unreachable code added
	}
	return n * 2
}`,
			expectEqual:         false, // Should detect the unreachable code change
			confidenceThreshold: 0.8,
		},
		{
			name: "Change in loop breaking condition - NOT equivalent",
			originalCode: `package main

func loopFunc() {
	for i := 0; i < 10; i++ {
		if i == 5 {
			break
		}
	}
}`,
			modifiedCode: `package main

func loopFunc() {
	for i := 0; i < 10; i++ {
		if i == 6 {  // Changed breaking condition
			break
		}
	}
}`,
			expectEqual:         false,
			confidenceThreshold: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewContentBasedCFGChecker(tt.confidenceThreshold, true)
			isEqual, report, err := checker.CheckEquivalence(tt.originalCode, tt.modifiedCode)

			require.NoError(t, err, "CFG equivalence check should not produce an error")

			if tt.expectEqual {
				assert.True(t, isEqual, "Expected code to be equivalent, but checker reported non-equivalence:\n%s", report)
			} else {
				assert.False(t, isEqual, "Expected code to be non-equivalent, but checker reported equivalence")
			}

			t.Logf("Equivalence report:\n%s", report)
		})
	}
}

func BenchmarkCFGEquivalence(b *testing.B) {
	smallCode := `package main

func simple(n int) bool {
	if n > 10 {
		return true
	} else {
		return false
	}
}`

	mediumCode := `package main

func medium(a, b int) int {
	result := 0
	if a > b {
		for i := 0; i < a; i++ {
			if i%2 == 0 {
				result += i
			} else {
				result -= i
			}
		}
	} else {
		for i := 0; i < b; i++ {
			if i%2 == 0 {
				result += i
			} else {
				result -= i
			}
		}
	}
	return result
}`

	largeCode := generateLargeFunction()

	benchmarks := []struct {
		name         string
		originalCode string
		modifiedCode string
	}{
		{
			name:         "Small function",
			originalCode: smallCode,
			modifiedCode: smallCode,
		},
		{
			name:         "Medium function",
			originalCode: mediumCode,
			modifiedCode: mediumCode,
		},
		{
			name:         "Large function",
			originalCode: largeCode,
			modifiedCode: largeCode,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			checker := NewContentBasedCFGChecker(0.8, false)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				isEqual, _, err := checker.CheckEquivalence(bm.originalCode, bm.modifiedCode)
				if err != nil || !isEqual {
					b.Fatalf("Expected equivalent CFGs, got error: %v, isEqual: %v", err, isEqual)
				}
			}
		})
	}
}

// Helper function to generate a large function for benchmarking
func generateLargeFunction() string {
	const funcTemplate = `package main

import (
	"errors"
	"fmt"
)

func largeFunction(values []int, threshold int) (int, error) {
	if len(values) == 0 {
		return 0, errors.New("empty input")
	}

	result := 0
	for i, v := range values {
		if v < 0 {
			continue
		}

		switch {
		case v > threshold * 2:
			result += v * 2
			if result > 1000 {
				return 1000, nil
			}
		case v > threshold:
			result += v
			if i > 0 && values[i-1] > threshold {
				result += values[i-1] / 2
			}
		default:
			result += v / 2
		}

		if i%10 == 0 {
			fmt.Println("Processing...")
		}
	}

	if result <= 0 {
		return 0, errors.New("no positive result")
	}

	for i := 0; i < 5; i++ {
		innerResult := 0
		for j := 0; j < 3; j++ {
			innerResult += i * j
			if innerResult > threshold {
				break
			}
		}
		result += innerResult
	}

	return result, nil
}
`
	return funcTemplate
}

func TestRangeStmtEquivalence(t *testing.T) {
	tests := []struct {
		name                string
		originalCode        string
		modifiedCode        string
		expectEqual         bool
		confidenceThreshold float64
	}{
		{
			name: "Equivalent range loops - same variables and expression",
			originalCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	for i, v := range nums {
		sum += v * i
	}
	return sum
}`,
			modifiedCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	for i, v := range nums {  // Added comment
		sum += v * i
	}
	return sum
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Different range key variable - NOT equivalent",
			originalCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	for i, v := range nums {
		sum += v * i
	}
	return sum
}`,
			modifiedCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	for idx, v := range nums {  // Changed key variable name
		sum += v * idx
	}
	return sum
}`,
			expectEqual:         false, // Changed variable name affects CFG
			confidenceThreshold: 0.8,
		},
		{
			name: "Different range value variable - NOT equivalent",
			originalCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	for i, v := range nums {
		sum += v * i
	}
	return sum
}`,
			modifiedCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	for i, val := range nums {  // Changed value variable name
		sum += val * i
	}
	return sum
}`,
			expectEqual:         false, // Changed variable name affects CFG
			confidenceThreshold: 0.8,
		},
		{
			name: "Different range collection - NOT equivalent",
			originalCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	for i, v := range nums {
		sum += v * i
	}
	return sum
}`,
			modifiedCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	otherSlice := nums[:len(nums)-1]  // Different slice
	for i, v := range otherSlice {  // Changed collection being iterated
		sum += v * i
	}
	return sum
}`,
			expectEqual:         false,
			confidenceThreshold: 0.8,
		},
		{
			name: "Key-only range to key-value range - NOT equivalent",
			originalCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	for i := range nums {
		sum += nums[i]
	}
	return sum
}`,
			modifiedCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	for i, v := range nums {  // Added value variable
		sum += v
	}
	return sum
}`,
			expectEqual:         false,
			confidenceThreshold: 0.8,
		},
		{
			name: "Value-only range loops (using _) - equivalent",
			originalCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	for _, v := range nums {
		sum += v
	}
	return sum
}`,
			modifiedCode: `package main

func rangeLoop(nums []int) int {
	sum := 0
	for _, v := range nums {
		sum += v
	}
	return sum
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Range over map - equivalent",
			originalCode: `package main

func mapRange(m map[string]int) int {
	sum := 0
	for k, v := range m {
		if k == "special" {
			sum += v * 2
		} else {
			sum += v
		}
	}
	return sum
}`,
			modifiedCode: `package main

func mapRange(m map[string]int) int {
	sum := 0
	for k, v := range m {  // Added comment
		if k == "special" {
			sum += v * 2
		} else {
			sum += v
		}
	}
	return sum
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Range over different map expression - NOT equivalent",
			originalCode: `package main

func mapRange(m map[string]int) int {
	sum := 0
	for k, v := range m {
		sum += v
	}
	return sum
}`,
			modifiedCode: `package main

func mapRange(m map[string]int) int {
	sum := 0
	filteredMap := make(map[string]int)
	for k, v := range m {
		if v > 0 {
			filteredMap[k] = v
		}
	}
	for k, v := range filteredMap {  // Different map expression
		sum += v
	}
	return sum
}`,
			expectEqual:         false,
			confidenceThreshold: 0.8,
		},
		{
			name: "Range over string - equivalent",
			originalCode: `package main

func countLetters(s string) int {
	count := 0
	for _, char := range s {
		if char >= 'a' && char <= 'z' {
			count++
		}
	}
	return count
}`,
			modifiedCode: `package main

func countLetters(s string) int {
	count := 0
	for _, char := range s {
		if char >= 'a' && char <= 'z' {
			count++
		}
	}
	return count
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Range over channel - equivalent",
			originalCode: `package main

func processChan(ch <-chan int) int {
	sum := 0
	for v := range ch {
		sum += v
	}
	return sum
}`,
			modifiedCode: `package main

func processChan(ch <-chan int) int {
	sum := 0
	for v := range ch {  // Added comment
		sum += v
	}
	return sum
}`,
			expectEqual:         true,
			confidenceThreshold: 0.8,
		},
		{
			name: "Range with break condition - NOT equivalent",
			originalCode: `package main

func rangeWithBreak(nums []int) int {
	sum := 0
	for i, v := range nums {
		if v > 100 {
			break
		}
		sum += v
	}
	return sum
}`,
			modifiedCode: `package main

func rangeWithBreak(nums []int) int {
	sum := 0
	for i, v := range nums {
		if v > 200 {  // Changed break condition
			break
		}
		sum += v
	}
	return sum
}`,
			expectEqual:         false,
			confidenceThreshold: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewContentBasedCFGChecker(tt.confidenceThreshold, true)
			isEqual, report, err := checker.CheckEquivalence(tt.originalCode, tt.modifiedCode)

			require.NoError(t, err, "CFG equivalence check should not produce an error")

			if tt.expectEqual {
				assert.True(t, isEqual, "Expected code to be equivalent, but checker reported non-equivalence:\n%s", report)
			} else {
				assert.False(t, isEqual, "Expected code to be non-equivalent, but checker reported equivalence")
			}

			t.Logf("Equivalence report:\n%s", report)
		})
	}
}
