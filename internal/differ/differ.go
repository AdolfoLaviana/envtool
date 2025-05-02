// Package differ provides functionality to compare two .env files
package differ

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// ComparisonResult represents the result of comparing two .env files
type ComparisonResult struct {
	OnlyInFirst     map[string]string
	OnlyInSecond    map[string]string
	DifferentValues map[string]ValueDifference
}

// ValueDifference represents a key with different values in two files
type ValueDifference struct {
	FirstValue  string
	SecondValue string
}

// DiffEnvFiles compares two .env files and displays the differences
func DiffEnvFiles(firstFilePath, secondFilePath string) {
	result, err := Compare(firstFilePath, secondFilePath)
	if err != nil {
		fmt.Printf("Error comparing .env files: %v\n", err)
		return
	}

	// Display results
	if len(result.OnlyInFirst) > 0 {
		fmt.Printf("\n🔍 Variables only in %s:\n", firstFilePath)
		printSortedEntries(result.OnlyInFirst)
	}

	if len(result.OnlyInSecond) > 0 {
		fmt.Printf("\n🔍 Variables only in %s:\n", secondFilePath)
		printSortedEntries(result.OnlyInSecond)
	}

	if len(result.DifferentValues) > 0 {
		fmt.Printf("\n🔄 Variables with different values:\n")
		printDifferences(result.DifferentValues)
	}

	// If no differences were found
	if len(result.OnlyInFirst) == 0 && len(result.OnlyInSecond) == 0 && len(result.DifferentValues) == 0 {
		fmt.Println("✅ Files are identical")
	} else {
		fmt.Println("\nTip: To merge the files, use 'envtool diff --merge <output_file> " + firstFilePath + " " + secondFilePath + "'")
	}
}

// Compare analyzes two .env files and returns their differences
func Compare(firstFilePath, secondFilePath string) (ComparisonResult, error) {
	result := ComparisonResult{
		OnlyInFirst:     make(map[string]string),
		OnlyInSecond:    make(map[string]string),
		DifferentValues: make(map[string]ValueDifference),
	}

	firstEntries, err := parseEnvFile(firstFilePath)
	if err != nil {
		return result, fmt.Errorf("failed to parse first file: %w", err)
	}

	secondEntries, err := parseEnvFile(secondFilePath)
	if err != nil {
		return result, fmt.Errorf("failed to parse second file: %w", err)
	}

	// Find entries only in first file and different values
	for key, firstValue := range firstEntries {
		if secondValue, exists := secondEntries[key]; exists {
			if firstValue != secondValue {
				result.DifferentValues[key] = ValueDifference{
					FirstValue:  firstValue,
					SecondValue: secondValue,
				}
			}
		} else {
			result.OnlyInFirst[key] = firstValue
		}
	}

	// Find entries only in second file
	for key, secondValue := range secondEntries {
		if _, exists := firstEntries[key]; !exists {
			result.OnlyInSecond[key] = secondValue
		}
	}

	return result, nil
}

// MergeFiles merges two .env files into a new one
func MergeFiles(firstFilePath, secondFilePath, outputFilePath string) error {
	result, err := Compare(firstFilePath, secondFilePath)
	if err != nil {
		return fmt.Errorf("failed to compare files: %w", err)
	}

	// Start with all entries from the first file
	mergedEntries, err := parseEnvFile(firstFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse first file: %w", err)
	}

	// Add entries that are only in the second file
	for key, value := range result.OnlyInSecond {
		mergedEntries[key] = value
	}

	// For entries with different values, keep the second file's value but comment the original
	for key, diff := range result.DifferentValues {
		mergedEntries[key] = diff.SecondValue
	}

	// Write the merged file
	file, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Sort keys for consistent output
	keys := make([]string, 0, len(mergedEntries))
	for key := range mergedEntries {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Write header
	fmt.Fprintf(file, "# Merged from %s and %s\n\n", firstFilePath, secondFilePath)

	// Write entries
	for _, key := range keys {
		value := mergedEntries[key]

		// If this key had different values, add a comment with original value
		if diff, isDiff := result.DifferentValues[key]; isDiff {
			fmt.Fprintf(file, "# Original value in %s: %s\n", firstFilePath, diff.FirstValue)
		}

		fmt.Fprintf(file, "%s=%s\n", key, value)
	}

	return nil
}

// parseEnvFile reads a .env file and returns a map of key-value pairs
func parseEnvFile(filePath string) (map[string]string, error) {
	entries := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		return entries, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineRegex := regexp.MustCompile(`^([A-Za-z0-9_]+)=(.*)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := lineRegex.FindStringSubmatch(line)
		if matches != nil {
			key := matches[1]
			value := matches[2]
			entries[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return entries, err
	}

	return entries, nil
}

// printSortedEntries prints entries in alphabetical order by key
func printSortedEntries(entries map[string]string) {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fmt.Printf("  %s=%s\n", key, entries[key])
	}
}

// printDifferences prints value differences in alphabetical order by key
func printDifferences(diffs map[string]ValueDifference) {
	keys := make([]string, 0, len(diffs))
	for key := range diffs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		diff := diffs[key]
		fmt.Printf("  %s:\n    - %s\n    + %s\n", key, diff.FirstValue, diff.SecondValue)
	}
}
