// Package generator provides functionality to generate .env files from templates/schemas
package generator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

// EnvSchema defines the structure of a schema configuration for .env files
type EnvSchema struct {
	Version     string              `json:"version"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Variables   map[string]Variable `json:"variables"`
	Sections    []Section           `json:"sections"`
}

// Variable defines the structure of an environment variable in the schema
type Variable struct {
	Description string      `json:"description"`
	Type        string      `json:"type"` // string, number, boolean, secret
	Default     interface{} `json:"default"`
	Required    bool        `json:"required"`
	Options     []string    `json:"options,omitempty"` // For enum type
}

// Section defines a logical grouping of variables
type Section struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Variables   []string `json:"variables"` // Names of variables in this section
}

// GenerateEnv generates a .env file from a schema
func GenerateEnv(schemaPath string) {
	if schemaPath == "" {
		fmt.Println("Error: Schema file path is required")
		fmt.Println("Usage: envtool generate --schema <schema_file.json>")
		return
	}

	// Read the schema file
	schema, err := readSchema(schemaPath)
	if err != nil {
		fmt.Printf("Error reading schema: %v\n", err)
		return
	}

	// Output file is .env by default
	outputFile := ".env"

	// Check if .env already exists, and if so, create a backup
	if _, err := os.Stat(outputFile); err == nil {
		backupFile := outputFile + ".bak"
		err = copyFile(outputFile, backupFile)
		if err != nil {
			fmt.Printf("Warning: Failed to create backup: %v\n", err)
		} else {
			fmt.Printf("Created backup of existing .env at %s\n", backupFile)
		}
	}

	// Generate the .env file
	err = generateFromSchema(schema, outputFile)
	if err != nil {
		fmt.Printf("Error generating .env file: %v\n", err)
		return
	}

	fmt.Printf("✅ Generated .env file from schema: %s\n", schemaPath)
}

// readSchema reads and parses a schema file
func readSchema(filePath string) (*EnvSchema, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var schema EnvSchema
	err = json.Unmarshal(data, &schema)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON schema: %w", err)
	}

	// If no sections are defined, create a default section with all variables
	if len(schema.Sections) == 0 {
		vars := make([]string, 0, len(schema.Variables))
		for v := range schema.Variables {
			vars = append(vars, v)
		}
		sort.Strings(vars)

		schema.Sections = []Section{
			{
				Name:        "Default",
				Description: "Environment Variables",
				Variables:   vars,
			},
		}
	}

	return &schema, nil
}

// generateFromSchema creates a .env file based on the schema
func generateFromSchema(schema *EnvSchema, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "# %s\n", schema.Name)
	if schema.Description != "" {
		fmt.Fprintf(file, "# %s\n", schema.Description)
	}
	fmt.Fprintf(file, "# Generated with envtool generate\n\n")

	// Write variables by section
	for _, section := range schema.Sections {
		fmt.Fprintf(file, "# %s\n", section.Name)
		if section.Description != "" {
			fmt.Fprintf(file, "# %s\n", section.Description)
		}
		fmt.Fprintln(file, "#")

		for _, varName := range section.Variables {
			variable, exists := schema.Variables[varName]
			if !exists {
				fmt.Fprintf(file, "# Warning: Variable %s referenced in section but not defined\n", varName)
				continue
			}

			// Add description as comment
			if variable.Description != "" {
				fmt.Fprintf(file, "# %s\n", variable.Description)
			}

			// Add default value or placeholder
			value := ""
			if variable.Default != nil {
				switch v := variable.Default.(type) {
				case string:
					value = v
				case bool:
					if v {
						value = "true"
					} else {
						value = "false"
					}
				case float64:
					value = fmt.Sprintf("%g", v)
				default:
					value = fmt.Sprintf("%v", v)
				}
			}

			if value == "" {
				if variable.Required {
					value = fmt.Sprintf("REQUIRED_%s", strings.ToUpper(variable.Type))
				} else {
					value = ""
				}
			}

			// If it's a secret type, add a note
			if variable.Type == "secret" && value != "" {
				fmt.Fprintln(file, "# Warning: This is a secret and should be kept secure")
				value = "REPLACE_WITH_ACTUAL_SECRET"
			}

			// Write the variable
			fmt.Fprintf(file, "%s=%s\n", varName, value)
			fmt.Fprintln(file, "")
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, data, 0644)
}

// CreateSampleSchema creates a sample schema file for demonstration purposes
func CreateSampleSchema(outputPath string) error {
	if outputPath == "" {
		outputPath = ".env.schema.json"
	}

	// Create a sample schema
	schema := EnvSchema{
		Version:     "1.0",
		Name:        "Sample Application Configuration",
		Description: "Configuration for a sample application",
		Variables: map[string]Variable{
			"APP_NAME": {
				Description: "Name of the application",
				Type:        "string",
				Default:     "My Application",
				Required:    true,
			},
			"APP_ENV": {
				Description: "Application environment",
				Type:        "string",
				Default:     "development",
				Required:    true,
				Options:     []string{"development", "staging", "production"},
			},
			"DEBUG": {
				Description: "Enable debug mode",
				Type:        "boolean",
				Default:     true,
				Required:    false,
			},
			"PORT": {
				Description: "Port to run the server on",
				Type:        "number",
				Default:     3000,
				Required:    true,
			},
			"DATABASE_URL": {
				Description: "Database connection string",
				Type:        "string",
				Default:     "postgresql://localhost:5432/myapp",
				Required:    true,
			},
			"API_KEY": {
				Description: "API key for external services",
				Type:        "secret",
				Default:     "",
				Required:    true,
			},
		},
		Sections: []Section{
			{
				Name:        "Application Settings",
				Description: "Basic application configuration",
				Variables:   []string{"APP_NAME", "APP_ENV", "DEBUG"},
			},
			{
				Name:        "Server Configuration",
				Description: "Server-related settings",
				Variables:   []string{"PORT"},
			},
			{
				Name:        "Database",
				Description: "Database connection settings",
				Variables:   []string{"DATABASE_URL"},
			},
			{
				Name:        "External Services",
				Description: "Configuration for external services",
				Variables:   []string{"API_KEY"},
			},
		},
	}

	// Convert to JSON
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Write to file
	err = ioutil.WriteFile(outputPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}

	return nil
}
