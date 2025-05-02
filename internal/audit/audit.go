// Package audit provides functionality to track and log changes to .env files
package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AuditRecord represents a record of a change to an .env file
type AuditRecord struct {
	Timestamp   time.Time          `json:"timestamp"`
	Hash        string             `json:"hash"`
	User        string             `json:"user"`
	ChangedVars map[string]VarDiff `json:"changedVars,omitempty"`
}

// VarDiff represents a change to a variable
type VarDiff struct {
	OldValue string `json:"oldValue,omitempty"`
	NewValue string `json:"newValue"`
}

// AuditLog represents the log of changes to an .env file
type AuditLog struct {
	FilePath string        `json:"filePath"`
	History  []AuditRecord `json:"history"`
}

// AuditEnv displays the audit log for an .env file
func AuditEnv(filePath string) {
	auditFile := getAuditFilePath(filePath)

	// Check if audit file exists
	if _, err := os.Stat(auditFile); os.IsNotExist(err) {
		fmt.Println("No audit history found.")
		fmt.Printf("Creating new audit log for %s\n", filePath)

		// Create a new audit entry for the current state
		err := createInitialAudit(filePath)
		if err != nil {
			fmt.Printf("Error creating audit log: %v\n", err)
		}
		return
	}

	// Load the audit log
	log, err := loadAuditLog(auditFile)
	if err != nil {
		fmt.Printf("Error loading audit log: %v\n", err)
		return
	}

	// Display the audit history
	fmt.Printf("Audit History for %s:\n\n", filePath)

	// Sort history from newest to oldest
	sort.Slice(log.History, func(i, j int) bool {
		return log.History[i].Timestamp.After(log.History[j].Timestamp)
	})

	for i, record := range log.History {
		fmt.Printf("%d. %s by %s\n", i+1, record.Timestamp.Format("2006-01-02 15:04:05"), record.User)
		fmt.Printf("   Hash: %s\n", record.Hash)

		if len(record.ChangedVars) > 0 {
			fmt.Println("   Changed variables:")

			// Sort variables by name for consistent output
			varNames := make([]string, 0, len(record.ChangedVars))
			for name := range record.ChangedVars {
				varNames = append(varNames, name)
			}
			sort.Strings(varNames)

			for _, name := range varNames {
				diff := record.ChangedVars[name]
				if diff.OldValue == "" {
					fmt.Printf("   + %s=%s (added)\n", name, diff.NewValue)
				} else if diff.NewValue == "" {
					fmt.Printf("   - %s=%s (removed)\n", name, diff.OldValue)
				} else {
					fmt.Printf("   * %s: %s → %s\n", name, diff.OldValue, diff.NewValue)
				}
			}
		} else if i == len(log.History)-1 {
			fmt.Println("   Initial version")
		} else {
			fmt.Println("   No changes detected")
		}

		fmt.Println()
	}
}

// RecordChange records a change to an .env file
func RecordChange(filePath string, oldVars, newVars map[string]string) error {
	auditFile := getAuditFilePath(filePath)

	var log AuditLog

	// Try to load existing audit log
	if _, err := os.Stat(auditFile); err == nil {
		log, err = loadAuditLog(auditFile)
		if err != nil {
			return fmt.Errorf("failed to load audit log: %w", err)
		}
	} else {
		// Initialize new log
		log = AuditLog{
			FilePath: filePath,
			History:  []AuditRecord{},
		}
	}

	// Calculate file hash
	hash, err := calculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Identify changed variables
	changedVars := identifyChanges(oldVars, newVars)

	// Create new audit record
	record := AuditRecord{
		Timestamp:   time.Now(),
		Hash:        hash,
		User:        getCurrentUser(),
		ChangedVars: changedVars,
	}

	// Add to history and save
	log.History = append(log.History, record)

	return saveAuditLog(log, auditFile)
}

// createInitialAudit creates an initial audit entry for a file
func createInitialAudit(filePath string) error {
	vars, err := parseEnvFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse .env file: %w", err)
	}

	// Calculate file hash
	hash, err := calculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Create initial audit record with all variables as "added"
	changedVars := make(map[string]VarDiff, len(vars))
	for name, value := range vars {
		changedVars[name] = VarDiff{
			NewValue: value,
		}
	}

	log := AuditLog{
		FilePath: filePath,
		History: []AuditRecord{
			{
				Timestamp:   time.Now(),
				Hash:        hash,
				User:        getCurrentUser(),
				ChangedVars: changedVars,
			},
		},
	}

	auditFile := getAuditFilePath(filePath)
	return saveAuditLog(log, auditFile)
}

// getAuditFilePath returns the path to the audit file for an .env file
func getAuditFilePath(envFilePath string) string {
	dir := filepath.Dir(envFilePath)
	base := filepath.Base(envFilePath)
	return filepath.Join(dir, "."+base+".audit.json")
}

// loadAuditLog loads an audit log from a file
func loadAuditLog(filePath string) (AuditLog, error) {
	var log AuditLog

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return log, err
	}

	err = json.Unmarshal(data, &log)
	return log, err
}

// saveAuditLog saves an audit log to a file
func saveAuditLog(log AuditLog, filePath string) error {
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, data, 0644)
}

// calculateFileHash calculates a SHA-256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// getCurrentUser gets the current user from the environment
func getCurrentUser() string {
	// Try different environment variables for different OS
	user := os.Getenv("USER") // Unix
	if user == "" {
		user = os.Getenv("USERNAME") // Windows
	}
	if user == "" {
		// Best effort to get hostname if username is not available
		hostname, err := os.Hostname()
		if err == nil {
			user = hostname
		} else {
			user = "unknown"
		}
	}
	return user
}

// identifyChanges identifies changes between two sets of variables
func identifyChanges(oldVars, newVars map[string]string) map[string]VarDiff {
	changes := make(map[string]VarDiff)

	// Find modified and deleted variables
	for name, oldValue := range oldVars {
		if newValue, exists := newVars[name]; exists {
			if oldValue != newValue {
				changes[name] = VarDiff{
					OldValue: oldValue,
					NewValue: newValue,
				}
			}
		} else {
			changes[name] = VarDiff{
				OldValue: oldValue,
				NewValue: "",
			}
		}
	}

	// Find added variables
	for name, newValue := range newVars {
		if _, exists := oldVars[name]; !exists {
			changes[name] = VarDiff{
				OldValue: "",
				NewValue: newValue,
			}
		}
	}

	return changes
}

// parseEnvFile parses an .env file into a map of variables
func parseEnvFile(filePath string) (map[string]string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	vars := make(map[string]string)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			vars[key] = value
		}
	}

	return vars, nil
}
