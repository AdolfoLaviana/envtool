// Package validator provides functionality to validate .env files
package validator

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Result represents the validation result for a .env file
type Result struct {
	InvalidLines  []InvalidLine
	DuplicateKeys []DuplicateKey
	MissingValues []string
	IsValid       bool
}

// InvalidLine represents a line that doesn't follow KEY=VALUE format
type InvalidLine struct {
	LineNumber int
	Content    string
	Reason     string
}

// DuplicateKey represents a key that appears multiple times in the file
type DuplicateKey struct {
	Key         string
	LineNumbers []int
}

// ValidateEnv validates the format and content of an .env file
// It checks for proper KEY=VALUE format, duplicate keys, and missing values
func ValidateEnv(filePath string) {
	result, err := Validate(filePath, "")
	if err != nil {
		fmt.Printf("Error validating .env file: %v\n", err)
		return
	}

	if result.IsValid {
		fmt.Println("✅ .env file is valid")
		return
	}

	// Display validation issues
	if len(result.InvalidLines) > 0 {
		fmt.Println("\n❌ Invalid format lines:")
		for _, invalid := range result.InvalidLines {
			fmt.Printf("  Line %d: %s (%s)\n", invalid.LineNumber, invalid.Content, invalid.Reason)
		}
	}

	if len(result.DuplicateKeys) > 0 {
		fmt.Println("\n⚠️ Duplicate keys:")
		for _, dup := range result.DuplicateKeys {
			fmt.Printf("  Key '%s' appears at lines %v\n", dup.Key, dup.LineNumbers)
		}
	}

	if len(result.MissingValues) > 0 {
		fmt.Println("\n⚠️ Missing or empty values:")
		for _, key := range result.MissingValues {
			fmt.Printf("  %s\n", key)
		}
	}
}

// Validate analyzes an .env file and returns a Result
// If baseFilePath is provided, it also checks for missing values compared to the base file
func Validate(filePath, baseFilePath string) (Result, error) {
	result := Result{
		InvalidLines:  make([]InvalidLine, 0),
		DuplicateKeys: make([]DuplicateKey, 0),
		MissingValues: make([]string, 0),
		IsValid:       true,
	}

	file, err := os.Open(filePath)
	if err != nil {
		return result, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	// Regular expression for KEY=VALUE format
	// Key can contain letters, numbers, and underscore
	validLineRegex := regexp.MustCompile(`^([A-Za-z0-9_]+)=(.*)$`)

	// Map to track keys for duplicate detection
	keys := make(map[string][]int)
	// Map to track keys with empty values
	emptyValues := make(map[string]bool)

	// Process each line
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if line has valid KEY=VALUE format
		matches := validLineRegex.FindStringSubmatch(line)
		if matches == nil {
			result.InvalidLines = append(result.InvalidLines, InvalidLine{
				LineNumber: lineNumber,
				Content:    line,
				Reason:     "does not match KEY=VALUE format",
			})
			result.IsValid = false
			continue
		}

		key := matches[1]
		value := matches[2]

		// Track key for duplicate detection
		keys[key] = append(keys[key], lineNumber)

		// Check for empty values
		if value == "" {
			emptyValues[key] = true
		}
	}

	// Check for duplicate keys
	for key, lines := range keys {
		if len(lines) > 1 {
			result.DuplicateKeys = append(result.DuplicateKeys, DuplicateKey{
				Key:         key,
				LineNumbers: lines,
			})
			result.IsValid = false
		}
	}

	// Check for empty values
	for key := range emptyValues {
		result.MissingValues = append(result.MissingValues, key)
		result.IsValid = false
	}

	// If baseFilePath is provided, check for missing keys compared to base
	if baseFilePath != "" {
		baseKeys, err := extractKeys(baseFilePath)
		if err == nil {
			for baseKey := range baseKeys {
				if _, exists := keys[baseKey]; !exists {
					result.MissingValues = append(result.MissingValues, baseKey)
					result.IsValid = false
				}
			}
		}
	}

	return result, nil
}

// extractKeys reads a .env file and returns a map of all keys
func extractKeys(filePath string) (map[string]bool, error) {
	keys := make(map[string]bool)

	file, err := os.Open(filePath)
	if err != nil {
		return keys, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	validLineRegex := regexp.MustCompile(`^([A-Za-z0-9_]+)=.*$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := validLineRegex.FindStringSubmatch(line)
		if matches != nil {
			keys[matches[1]] = true
		}
	}

	return keys, nil
}
