package cleanup

import (
	"testing"
)

func TestCleanCodeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "fix if keyword",
			input:    "lf (x > 0) {",
			expected: "if (x > 0) {",
		},
		{
			name:     "fix return keyword",
			input:    "retum nil",
			expected: "return nil",
		},
		{
			name:     "fix function keyword",
			input:    "functi0n hello() {",
			expected: "function hello() {",
		},
		{
			name:     "fix class keyword",
			input:    "c1ass MyClass {",
			expected: "class MyClass {",
		},
		{
			name:     "fix false/null",
			input:    "if x == fa1se || y == nul1 {",
			expected: "if x == false || y == null {",
		},
		{
			name:     "preserve correct code",
			input:    "func main() {\n\tfmt.Println(\"hello\")\n}",
			expected: "func main() {\n\tfmt.Println(\"hello\")\n}",
		},
		{
			name:     "fix arrow operator",
			input:    "const fn =2 () =2 42",
			expected: "const fn => () => 42",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CleanCodeText(tc.input)
			if result != tc.expected {
				t.Errorf("got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestRemoveLineNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		hasNums  bool
		expected string
	}{
		{
			name: "with line numbers",
			input: `  1    func main() {
  2        fmt.Println("hello")
  3    }`,
			hasNums: true,
			expected: `func main() {
    fmt.Println("hello")
}`,
		},
		{
			name: "without line numbers",
			input: `func main() {
    fmt.Println("hello")
}`,
			hasNums: false,
			expected: `func main() {
    fmt.Println("hello")
}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := RemoveLineNumbers(tc.input)
			if result != tc.expected {
				t.Errorf("got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestNormalizeIndentation(t *testing.T) {
	input := "func main() {\n\tfmt.Println()\n}"
	expected := "func main() {\n    fmt.Println()\n}"
	result := NormalizeIndentation(input, 4)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}
