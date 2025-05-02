// Package linter provides functionality to lint and fix .env files
package linter

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
)

// LintResult represents the result of linting an .env file
type LintResult struct {
	BadNames    []BadName
	Suggestions []Suggestion
	IsValid     bool
}

// BadName represents a variable name that doesn't follow naming conventions
type BadName struct {
	LineNumber int
	Name       string
	Suggestion string
	Reason     string
}

// Suggestion represents a suggested change to an .env file
type Suggestion struct {
	Type          string // "rename", "reorder", "add", "remove", "format"
	LineNumber    int
	CurrentText   string
	SuggestedText string
	Reason        string
}

// LintEnv lints an .env file and optionally fixes issues
func LintEnv(filePath string, fix bool) {
	result, err := Lint(filePath)
	if err != nil {
		fmt.Printf("Error linting .env file: %v\n", err)
		return
	}

	if result.IsValid && len(result.Suggestions) == 0 {
		fmt.Println("✅ .env file follows best practices")
		return
	}

	// Display linting issues
	if len(result.BadNames) > 0 {
		fmt.Println("\n⚠️ Variable names not following conventions:")
		for _, bad := range result.BadNames {
			fmt.Printf("  Line %d: %s → %s (%s)\n",
				bad.LineNumber, bad.Name, bad.Suggestion, bad.Reason)
		}
	}

	if len(result.Suggestions) > 0 {
		fmt.Println("\n💡 Suggestions:")
		for _, suggestion := range result.Suggestions {
			fmt.Printf("  [%s] Line %d: ", suggestion.Type, suggestion.LineNumber)

			switch suggestion.Type {
			case "rename":
				fmt.Printf("Rename '%s' to '%s'\n",
					suggestion.CurrentText, suggestion.SuggestedText)
			case "reorder":
				fmt.Println("Reorder variables alphabetically")
			case "format":
				fmt.Printf("Format: '%s' → '%s'\n",
					suggestion.CurrentText, suggestion.SuggestedText)
			case "add":
				fmt.Printf("Add comment: '%s'\n", suggestion.SuggestedText)
			}

			if suggestion.Reason != "" {
				fmt.Printf("    Reason: %s\n", suggestion.Reason)
			}
		}
	}

	// Apply fixes if requested
	if fix {
		err = ApplyFixes(filePath, result)
		if err != nil {
			fmt.Printf("Error applying fixes: %v\n", err)
			return
		}
		fmt.Println("\n✅ Applied suggested fixes to the .env file")
	} else if !result.IsValid || len(result.Suggestions) > 0 {
		fmt.Println("\nTip: Use 'envtool lint --fix " + filePath + "' to apply these suggestions")
	}
}

// Lint analyzes an .env file and returns a LintResult
func Lint(filePath string) (LintResult, error) {
	result := LintResult{
		BadNames:    make([]BadName, 0),
		Suggestions: make([]Suggestion, 0),
		IsValid:     true,
	}

	file, err := os.Open(filePath)
	if err != nil {
		return result, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Regular expression for KEY=VALUE format
	validLineRegex := regexp.MustCompile(`^([A-Za-z0-9_]+)=(.*)$`)

	// Regular expression for conventional environment variable naming (UPPER_SNAKE_CASE)
	conventionalNameRegex := regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

	// Track sections for potential reordering
	sections := make(map[string][]string)
	currentSection := "default"
	sectionOrder := []string{"default"}

	// Track line numbers of variables for potential reordering
	varLines := make(map[string]int)

	// Track names to check for alphabetical order
	names := make([]string, 0)

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Check for section comments
		if strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "# ") {
			sectionName := strings.TrimPrefix(trimmedLine, "#")
			sectionName = strings.TrimSpace(sectionName)

			if sectionName != "" && sectionName != currentSection {
				currentSection = sectionName
				if _, exists := sections[currentSection]; !exists {
					sections[currentSection] = make([]string, 0)
					sectionOrder = append(sectionOrder, currentSection)
				}
			}
			continue
		}

		// Skip empty lines and regular comments
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Check if line has valid KEY=VALUE format
		matches := validLineRegex.FindStringSubmatch(trimmedLine)
		if matches == nil {
			result.IsValid = false
			continue
		}

		name := matches[1]
		names = append(names, name)
		varLines[name] = lineNumber
		sections[currentSection] = append(sections[currentSection], name)

		// Check naming convention
		if !conventionalNameRegex.MatchString(name) {
			suggestion := toConventionalName(name)
			result.BadNames = append(result.BadNames, BadName{
				LineNumber: lineNumber,
				Name:       name,
				Suggestion: suggestion,
				Reason:     "Names should be UPPER_SNAKE_CASE",
			})

			result.Suggestions = append(result.Suggestions, Suggestion{
				Type:          "rename",
				LineNumber:    lineNumber,
				CurrentText:   name,
				SuggestedText: suggestion,
				Reason:        "Names should be UPPER_SNAKE_CASE",
			})

			result.IsValid = false
		}

		// Check for common prefixes that should be grouped
		if containsPrefix(name, "DB_", "DATABASE_") && !strings.HasPrefix(name, "DB_") && !strings.HasPrefix(name, "DATABASE_") {
			suggestion := "DB_" + name
			result.Suggestions = append(result.Suggestions, Suggestion{
				Type:          "rename",
				LineNumber:    lineNumber,
				CurrentText:   name,
				SuggestedText: suggestion,
				Reason:        "Database-related variables should have DB_ prefix",
			})
		}

		// Check formatting of the line
		formattedLine := fmt.Sprintf("%s=%s", name, matches[2])
		if formattedLine != trimmedLine {
			result.Suggestions = append(result.Suggestions, Suggestion{
				Type:          "format",
				LineNumber:    lineNumber,
				CurrentText:   trimmedLine,
				SuggestedText: formattedLine,
				Reason:        "Consistent formatting improves readability",
			})
		}
	}

	// Check for alphabetical order within sections
	for section, vars := range sections {
		if !isAlphabeticallySorted(vars) {
			result.Suggestions = append(result.Suggestions, Suggestion{
				Type:   "reorder",
				Reason: fmt.Sprintf("Variables in section '%s' should be alphabetically sorted", section),
			})
		}
	}

	return result, nil
}

// ApplyFixes applies the suggested fixes to an .env file
func ApplyFixes(filePath string, result LintResult) error {
	// Read the entire file
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Apply rename and format suggestions
	for _, suggestion := range result.Suggestions {
		if suggestion.Type == "rename" || suggestion.Type == "format" {
			lineIdx := suggestion.LineNumber - 1
			if lineIdx >= 0 && lineIdx < len(lines) {
				line := lines[lineIdx]

				if suggestion.Type == "rename" {
					// Replace the variable name while preserving the value
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						lines[lineIdx] = suggestion.SuggestedText + "=" + parts[1]
					}
				} else if suggestion.Type == "format" {
					lines[lineIdx] = suggestion.SuggestedText
				}
			}
		}
	}

	// Handle reordering if suggested
	for _, suggestion := range result.Suggestions {
		if suggestion.Type == "reorder" {
			lines = reorderLines(lines)
			break
		}
	}

	// Write the modified content back to the file
	err = ioutil.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return fmt.Errorf("failed to write changes: %w", err)
	}

	return nil
}

// toConventionalName converts a variable name to UPPER_SNAKE_CASE
func toConventionalName(name string) string {
	// Convert to uppercase
	name = strings.ToUpper(name)

	// Replace non-conventional separators with underscores
	separatorRegex := regexp.MustCompile(`[- .]`)
	name = separatorRegex.ReplaceAllString(name, "_")

	// Remove consecutive underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	return name
}

// containsPrefix checks if a string contains any of the given words
func containsPrefix(s string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		trimmed := strings.TrimPrefix(prefix, "_")
		if strings.Contains(strings.ToUpper(s), strings.ToUpper(trimmed)) {
			return true
		}
	}
	return false
}

// isAlphabeticallySorted checks if a slice of strings is alphabetically sorted
func isAlphabeticallySorted(strs []string) bool {
	for i := 1; i < len(strs); i++ {
		if strs[i-1] > strs[i] {
			return false
		}
	}
	return true
}

// reorderLines reorders variables in a .env file by sections
func reorderLines(lines []string) []string {
	// Identify sections and their variables
	sections := make(map[string][]string)
	sectionOrder := []string{}
	variables := make(map[string]string)

	currentSection := "default"
	sectionOrder = append(sectionOrder, currentSection)

	// First pass: identify sections and collect variables
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			continue
		}

		// Check for section comments
		if strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "# ") {
			sectionName := strings.TrimPrefix(trimmedLine, "#")
			sectionName = strings.TrimSpace(sectionName)

			if sectionName != "" && sectionName != currentSection {
				currentSection = sectionName
				if _, exists := sections[currentSection]; !exists {
					sections[currentSection] = make([]string, 0)
					sectionOrder = append(sectionOrder, currentSection)
				}
			}
			continue
		}

		// Extract variable name if it's a variable line
		parts := strings.SplitN(trimmedLine, "=", 2)
		if len(parts) == 2 && !strings.HasPrefix(trimmedLine, "#") {
			name := strings.TrimSpace(parts[0])
			sections[currentSection] = append(sections[currentSection], name)
			variables[name] = trimmedLine
		}
	}

	// Sort variables within each section
	for section := range sections {
		sort.Strings(sections[section])
	}

	// Rebuild the file content
	var result []string
	seenBlankLine := false

	for _, section := range sectionOrder {
		if len(sections[section]) == 0 {
			continue
		}

		// Add blank line between sections if not already there
		if !seenBlankLine && len(result) > 0 {
			result = append(result, "")
		}

		if section != "default" {
			result = append(result, "#"+section)
		}

		// Add sorted variables
		for _, name := range sections[section] {
			if line, exists := variables[name]; exists {
				result = append(result, line)
			}
		}

		seenBlankLine = false
	}

	return result
}
