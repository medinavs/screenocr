package cleanup

import (
	"regexp"
	"strings"
)

// CleanCodeText applies heuristic fixes for common OCR mistakes in code.
func CleanCodeText(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		line = fixCommonMistakes(line)
		line = fixBrackets(line)
		line = fixOperators(line)
		result = append(result, line)
	}

	output := strings.Join(result, "\n")
	output = regexp.MustCompile(`\n{4,}`).ReplaceAllString(output, "\n\n\n")

	return strings.TrimSpace(output)
}

func fixCommonMistakes(line string) string {
	// Fix common l/1/I confusion in code contexts

	// "lf" at line start likely means "if"
	if strings.HasPrefix(strings.TrimSpace(line), "lf ") || strings.HasPrefix(strings.TrimSpace(line), "lf(") {
		line = strings.Replace(line, "lf ", "if ", 1)
		line = strings.Replace(line, "lf(", "if(", 1)
	}

	replacements := map[string]string{
		"retum":    "return",
		"retura":   "return",
		"functi0n": "function",
		"funct1on": "function",
		"c1ass":    "class",
		"publ1c":   "public",
		"pr1vate":  "private",
		"stati¢":   "static",
		"v0id":     "void",
		"nul1":     "null",
		"fa1se":    "false",
		"tru3":     "true",
		"e1se":     "else",
		"whi1e":    "while",
		"imp0rt":   "import",
		"pr1nt":    "print",
		"str1ng":   "string",
		"1nt":      "int",
	}

	for wrong, right := range replacements {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(wrong) + `\b`)
		line = re.ReplaceAllString(line, right)
	}

	return line
}

func fixBrackets(line string) string {
	line = strings.ReplaceAll(line, "{{", "{{") // Already correct, no-op
	line = strings.ReplaceAll(line, "}}", "}}") // Already correct, no-op

	line = strings.ReplaceAll(line, "¢", "(")

	return line
}

func fixOperators(line string) string {
	// "=>" sometimes becomes "=2" or "=>"
	line = strings.ReplaceAll(line, "=2", "=>")

	// "!=" sometimes becomes "1="
	re := regexp.MustCompile(`([^a-zA-Z0-9])1=`)
	line = re.ReplaceAllString(line, "${1}!=")

	return line
}

// NormalizeIndentation converts mixed spaces/tabs to consistent indentation.
func NormalizeIndentation(text string, tabWidth int) string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if len(trimmed) == 0 {
			result = append(result, "")
			continue
		}

		prefix := line[:len(line)-len(trimmed)]
		expanded := strings.ReplaceAll(prefix, "\t", strings.Repeat(" ", tabWidth))
		result = append(result, expanded+trimmed)
	}

	return strings.Join(result, "\n")
}

// RemoveLineNumbers strips leading line numbers that OCR might pick up
// from code editor screenshots.
func RemoveLineNumbers(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	lineNumPattern := regexp.MustCompile(`^\s*\d{1,5}\s{1,4}`)
	matchCount := 0
	for _, line := range lines {
		if lineNumPattern.MatchString(line) {
			matchCount++
		}
	}

	// Only strip if >60% of lines have line numbers
	if float64(matchCount)/float64(len(lines)) < 0.6 {
		return text
	}

	for _, line := range lines {
		result = append(result, lineNumPattern.ReplaceAllString(line, ""))
	}

	return strings.Join(result, "\n")
}
