// Package ui provides a graphical user interface for the envtool application
package ui

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/adolfoc/envtool/internal/audit"
	"github.com/adolfoc/envtool/internal/crypto"
	"github.com/adolfoc/envtool/internal/differ"
	"github.com/adolfoc/envtool/internal/generator"
	"github.com/adolfoc/envtool/internal/linter"
	"github.com/adolfoc/envtool/internal/sync"
	"github.com/adolfoc/envtool/internal/validator"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// StartUI launches the graphical user interface or falls back to command-line mode
func StartUI() {
	// Check if we can initialize GUI or need to fall back to CLI
	if err := initFyneUI(); err != nil {
		fmt.Println("Error initializing GUI:", err)
		fmt.Println("This could be due to missing OpenGL dependencies on your system.")
		fmt.Println("\nTo fix this issue on Windows:")
		fmt.Println("1. Install MinGW-w64 or MSYS2 with gcc compiler")
		fmt.Println("2. Install the required dependencies for OpenGL development")
		fmt.Println("3. Make sure CGO_ENABLED=1 is set when building the application")

		fmt.Println("\nIn the meantime, you can use the command-line interface:")
		fmt.Println("envtool help - to see available commands")
		return
	}
}

// initFyneUI initializes the Fyne UI in a separate function to handle build errors
func initFyneUI() error {
	return fyneUIStarter()
}

// fyneUIStarter launches the graphical user interface
func fyneUIStarter() error {
	// Initialize Fyne application
	a := app.New()
	w := a.NewWindow("EnvTool - Environment Management")
	w.Resize(fyne.NewSize(800, 600))

	// Create tabs for different functionalities
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Validate", theme.DocumentValidIcon(), createValidateTab(w)),
		container.NewTabItemWithIcon("Compare", theme.ContentDiffIcon(), createCompareTab(w)),
		container.NewTabItemWithIcon("Encrypt", theme.SecurityIcon(), createEncryptTab(w)),
		container.NewTabItemWithIcon("Sync", theme.SyncIcon(), createSyncTab(w)),
		container.NewTabItemWithIcon("Generate", theme.FileApplicationIcon(), createGenerateTab(w)),
		container.NewTabItemWithIcon("Audit", theme.HistoryIcon(), createAuditTab(w)),
		container.NewTabItemWithIcon("Lint", theme.DocumentCheckIcon(), createLintTab(w)),
		container.NewTabItemWithIcon("About", theme.InfoIcon(), createAboutTab()),
	)

	tabs.SetTabLocation(container.TabLocationLeading)

	// Set window content and show
	w.SetContent(tabs)
	w.ShowAndRun()
	return nil
}

// createValidateTab creates the tab for validating .env files
func createValidateTab(window fyne.Window) fyne.CanvasObject {
	// File selection
	filePath := widget.NewEntry()
	filePath.SetPlaceHolder("Path to .env file")

	// Base file selection (optional)
	baseFilePath := widget.NewEntry()
	baseFilePath.SetPlaceHolder("Optional: Path to base .env file for comparison")

	// Browse button
	browseBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if uri == nil {
				return
			}
			filePath.SetText(uri.URI().Path())
		}, window)
	})

	browseBtnBase := widget.NewButton("Browse Base...", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if uri == nil {
				return
			}
			baseFilePath.SetText(uri.URI().Path())
		}, window)
	})

	// Results display
	results := widget.NewMultiLineEntry()
	results.SetPlaceHolder("Validation results will appear here")
	results.Disable()

	// Validate button
	validateBtn := widget.NewButton("Validate", func() {
		if filePath.Text == "" {
			dialog.ShowInformation("Error", "Please select a file to validate", window)
			return
		}

		// Capture console output
		output := captureOutput(func() {
			result, err := validator.Validate(filePath.Text, baseFilePath.Text)
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
		})

		results.SetText(output)
	})

	// Layout
	fileSelectBox := container.NewHBox(filePath, browseBtn)
	baseFileSelectBox := container.NewHBox(baseFilePath, browseBtnBase)

	return container.NewVBox(
		widget.NewLabel("Validate .env File Format"),
		widget.NewLabel("Select file to validate:"),
		fileSelectBox,
		widget.NewLabel("Optional: Select base file to check for missing keys:"),
		baseFileSelectBox,
		validateBtn,
		widget.NewLabel("Results:"),
		container.NewMax(results),
	)
}

// createCompareTab creates the tab for comparing .env files
func createCompareTab(window fyne.Window) fyne.CanvasObject {
	// File selection
	filePath1 := widget.NewEntry()
	filePath1.SetPlaceHolder("Path to first .env file")

	filePath2 := widget.NewEntry()
	filePath2.SetPlaceHolder("Path to second .env file")

	outputPath := widget.NewEntry()
	outputPath.SetPlaceHolder("Path to save merged output (optional)")

	// Browse buttons
	browseBtn1 := widget.NewButton("Browse...", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if uri == nil {
				return
			}
			filePath1.SetText(uri.URI().Path())
		}, window)
	})

	browseBtn2 := widget.NewButton("Browse...", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if uri == nil {
				return
			}
			filePath2.SetText(uri.URI().Path())
		}, window)
	})

	browseOutputBtn := widget.NewButton("Browse...", func() {
		dialog.ShowSaveFileDialog(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if writer == nil {
				return
			}
			outputPath.SetText(writer.URI().Path())
			writer.Close()
		}, window)
	})

	// Results display
	results := widget.NewMultiLineEntry()
	results.SetPlaceHolder("Comparison results will appear here")
	results.Disable()

	// Compare button
	compareBtn := widget.NewButton("Compare", func() {
		if filePath1.Text == "" || filePath2.Text == "" {
			dialog.ShowInformation("Error", "Please select both files to compare", window)
			return
		}

		// Capture console output
		output := captureOutput(func() {
			result, err := differ.Compare(filePath1.Text, filePath2.Text)
			if err != nil {
				fmt.Printf("Error comparing .env files: %v\n", err)
				return
			}

			// Display results
			if len(result.OnlyInFirst) > 0 {
				fmt.Printf("\n🔍 Variables only in %s:\n", filePath1.Text)
				for key, value := range result.OnlyInFirst {
					fmt.Printf("  %s=%s\n", key, value)
				}
			}

			if len(result.OnlyInSecond) > 0 {
				fmt.Printf("\n🔍 Variables only in %s:\n", filePath2.Text)
				for key, value := range result.OnlyInSecond {
					fmt.Printf("  %s=%s\n", key, value)
				}
			}

			if len(result.DifferentValues) > 0 {
				fmt.Printf("\n🔄 Variables with different values:\n")
				for key, diff := range result.DifferentValues {
					fmt.Printf("  %s:\n    - %s\n    + %s\n", key, diff.FirstValue, diff.SecondValue)
				}
			}

			// If no differences were found
			if len(result.OnlyInFirst) == 0 && len(result.OnlyInSecond) == 0 && len(result.DifferentValues) == 0 {
				fmt.Println("✅ Files are identical")
			}
		})

		results.SetText(output)
	})

	// Merge button
	mergeBtn := widget.NewButton("Merge", func() {
		if filePath1.Text == "" || filePath2.Text == "" {
			dialog.ShowInformation("Error", "Please select both files to merge", window)
			return
		}

		if outputPath.Text == "" {
			dialog.ShowInformation("Error", "Please specify output file for merge", window)
			return
		}

		err := differ.MergeFiles(filePath1.Text, filePath2.Text, outputPath.Text)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		dialog.ShowInformation("Success", "Files merged successfully", window)
	})

	// Layout
	fileSelectBox1 := container.NewHBox(filePath1, browseBtn1)
	fileSelectBox2 := container.NewHBox(filePath2, browseBtn2)
	outputSelectBox := container.NewHBox(outputPath, browseOutputBtn)
	buttonsBox := container.NewHBox(compareBtn, mergeBtn)

	return container.NewVBox(
		widget.NewLabel("Compare .env Files"),
		widget.NewLabel("Select first .env file:"),
		fileSelectBox1,
		widget.NewLabel("Select second .env file:"),
		fileSelectBox2,
		widget.NewLabel("Output file for merge (optional):"),
		outputSelectBox,
		buttonsBox,
		widget.NewLabel("Results:"),
		container.NewMax(results),
	)
}

// createEncryptTab creates the tab for encrypting and decrypting .env files
func createEncryptTab(window fyne.Window) fyne.CanvasObject {
	// File selection
	filePath := widget.NewEntry()
	filePath.SetPlaceHolder("Path to .env file")

	outputPath := widget.NewEntry()
	outputPath.SetPlaceHolder("Output file path (optional)")

	// Key input
	keyEntry := widget.NewPasswordEntry()
	keyEntry.SetPlaceHolder("Encryption key (leave empty to use environment variable or generate)")

	// Browse buttons
	browseBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if uri == nil {
				return
			}
			filePath.SetText(uri.URI().Path())
		}, window)
	})

	browseOutputBtn := widget.NewButton("Browse...", func() {
		dialog.ShowSaveFileDialog(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if writer == nil {
				return
			}
			outputPath.SetText(writer.URI().Path())
			writer.Close()
		}, window)
	})

	// Results display
	results := widget.NewMultiLineEntry()
	results.SetPlaceHolder("Operation results will appear here")
	results.Disable()

	// Encrypt button
	encryptBtn := widget.NewButton("Encrypt", func() {
		if filePath.Text == "" {
			dialog.ShowInformation("Error", "Please select a file to encrypt", window)
			return
		}

		// Set encryption key if provided
		if keyEntry.Text != "" {
			os.Setenv(crypto.DefaultKeyEnvVar, keyEntry.Text)
		}

		// Capture console output
		output := captureOutput(func() {
			crypto.EncryptEnv(filePath.Text, outputPath.Text)
		})

		results.SetText(output)
	})

	// Decrypt button
	decryptBtn := widget.NewButton("Decrypt", func() {
		if filePath.Text == "" {
			dialog.ShowInformation("Error", "Please select a file to decrypt", window)
			return
		}

		// Set encryption key if provided
		if keyEntry.Text != "" {
			os.Setenv(crypto.DefaultKeyEnvVar, keyEntry.Text)
		}

		// Capture console output
		output := captureOutput(func() {
			crypto.DecryptEnv(filePath.Text, outputPath.Text)
		})

		results.SetText(output)
	})

	// Layout
	fileSelectBox := container.NewHBox(filePath, browseBtn)
	outputSelectBox := container.NewHBox(outputPath, browseOutputBtn)
	buttonsBox := container.NewHBox(encryptBtn, decryptBtn)

	return container.NewVBox(
		widget.NewLabel("Encrypt/Decrypt .env Files"),
		widget.NewLabel("Select file:"),
		fileSelectBox,
		widget.NewLabel("Output file (optional):"),
		outputSelectBox,
		widget.NewLabel("Encryption key:"),
		keyEntry,
		buttonsBox,
		widget.NewLabel("Results:"),
		container.NewMax(results),
	)
}

// createSyncTab creates the tab for remote synchronization
func createSyncTab(window fyne.Window) fyne.CanvasObject {
	// File selection
	filePath := widget.NewEntry()
	filePath.SetPlaceHolder("Path to .env file")

	// Environment selection
	envSelect := widget.NewSelect([]string{"production", "staging", "development", "testing"}, nil)
	envSelect.SetSelected("production")

	// Provider selection
	providerSelect := widget.NewSelect([]string{"s3", "firebase", "git"}, nil)
	providerSelect.SetSelected("s3")

	// Browse button
	browseBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if uri == nil {
				return
			}
			filePath.SetText(uri.URI().Path())
		}, window)
	})

	// Results display
	results := widget.NewMultiLineEntry()
	results.SetPlaceHolder("Operation results will appear here")
	results.Disable()

	// Configuration fields
	configFields := widget.NewForm()

	// S3 config
	s3BucketEntry := widget.NewEntry()
	s3BucketEntry.SetPlaceHolder("S3 Bucket Name")
	s3RegionEntry := widget.NewEntry()
	s3RegionEntry.SetPlaceHolder("AWS Region")
	s3RegionEntry.SetText("us-east-1")

	// Firebase config
	firebaseProjectEntry := widget.NewEntry()
	firebaseProjectEntry.SetPlaceHolder("Firebase Project ID")
	firebasePathEntry := widget.NewEntry()
	firebasePathEntry.SetPlaceHolder("Firebase Storage Path")
	firebasePathEntry.SetText("envtool")

	// Git config
	gitRepoEntry := widget.NewEntry()
	gitRepoEntry.SetPlaceHolder("Git Repository URL")
	gitBranchEntry := widget.NewEntry()
	gitBranchEntry.SetPlaceHolder("Git Branch")
	gitBranchEntry.SetText("main")

	// Show config fields based on provider selection
	updateConfigFields := func() {
		configFields.Items = []*widget.FormItem{}

		switch providerSelect.Selected {
		case "s3":
			configFields.Append("Bucket Name", s3BucketEntry)
			configFields.Append("Region", s3RegionEntry)
		case "firebase":
			configFields.Append("Project ID", firebaseProjectEntry)
			configFields.Append("Storage Path", firebasePathEntry)
		case "git":
			configFields.Append("Repository URL", gitRepoEntry)
			configFields.Append("Branch", gitBranchEntry)
		}

		configFields.Refresh()
	}

	providerSelect.OnChanged = func(s string) {
		updateConfigFields()
	}

	// Initial config fields setup
	updateConfigFields()

	// Push button
	pushBtn := widget.NewButton("Push", func() {
		if filePath.Text == "" {
			dialog.ShowInformation("Error", "Please select a file to push", window)
			return
		}

		// Set environment variables for configuration
		switch providerSelect.Selected {
		case "s3":
			os.Setenv("ENVTOOL_SYNC_PROVIDER", "s3")
			os.Setenv("ENVTOOL_S3_BUCKET", s3BucketEntry.Text)
			os.Setenv("ENVTOOL_S3_REGION", s3RegionEntry.Text)
		case "firebase":
			os.Setenv("ENVTOOL_SYNC_PROVIDER", "firebase")
			os.Setenv("ENVTOOL_FIREBASE_PROJECT", firebaseProjectEntry.Text)
			os.Setenv("ENVTOOL_FIREBASE_PATH", firebasePathEntry.Text)
		case "git":
			os.Setenv("ENVTOOL_SYNC_PROVIDER", "git")
			os.Setenv("ENVTOOL_GIT_REPO", gitRepoEntry.Text)
			os.Setenv("ENVTOOL_GIT_BRANCH", gitBranchEntry.Text)
		}

		// Capture console output
		output := captureOutput(func() {
			sync.PushEnv(filePath.Text, envSelect.Selected)
		})

		results.SetText(output)
	})

	// Pull button
	pullBtn := widget.NewButton("Pull", func() {
		// Set environment variables for configuration
		switch providerSelect.Selected {
		case "s3":
			os.Setenv("ENVTOOL_SYNC_PROVIDER", "s3")
			os.Setenv("ENVTOOL_S3_BUCKET", s3BucketEntry.Text)
			os.Setenv("ENVTOOL_S3_REGION", s3RegionEntry.Text)
		case "firebase":
			os.Setenv("ENVTOOL_SYNC_PROVIDER", "firebase")
			os.Setenv("ENVTOOL_FIREBASE_PROJECT", firebaseProjectEntry.Text)
			os.Setenv("ENVTOOL_FIREBASE_PATH", firebasePathEntry.Text)
		case "git":
			os.Setenv("ENVTOOL_SYNC_PROVIDER", "git")
			os.Setenv("ENVTOOL_GIT_REPO", gitRepoEntry.Text)
			os.Setenv("ENVTOOL_GIT_BRANCH", gitBranchEntry.Text)
		}

		// Capture console output
		output := captureOutput(func() {
			sync.PullEnv(envSelect.Selected)
		})

		results.SetText(output)
	})

	// Layout
	fileSelectBox := container.NewHBox(filePath, browseBtn)
	buttonsBox := container.NewHBox(pushBtn, pullBtn)

	return container.NewVBox(
		widget.NewLabel("Remote Synchronization"),
		widget.NewLabel("Select file to push (not needed for pull):"),
		fileSelectBox,
		widget.NewLabel("Environment:"),
		envSelect,
		widget.NewLabel("Storage Provider:"),
		providerSelect,
		widget.NewLabel("Provider Configuration:"),
		configFields,
		buttonsBox,
		widget.NewLabel("Results:"),
		container.NewMax(results),
	)
}

// createGenerateTab creates the tab for template generation
func createGenerateTab(window fyne.Window) fyne.CanvasObject {
	// Schema file selection
	schemaPath := widget.NewEntry()
	schemaPath.SetPlaceHolder("Path to schema file")

	// Output file selection
	outputPath := widget.NewEntry()
	outputPath.SetPlaceHolder("Output file path (defaults to .env)")

	// Create sample schema checkbox
	createSampleCheck := widget.NewCheck("Create sample schema file", nil)

	// Browse buttons
	browseSchemaBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if uri == nil {
				return
			}
			schemaPath.SetText(uri.URI().Path())
		}, window)
	})

	browseOutputBtn := widget.NewButton("Browse...", func() {
		dialog.ShowSaveFileDialog(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if writer == nil {
				return
			}
			outputPath.SetText(writer.URI().Path())
			writer.Close()
		}, window)
	})

	// Results display
	results := widget.NewMultiLineEntry()
	results.SetPlaceHolder("Operation results will appear here")
	results.Disable()

	// Generate button
	generateBtn := widget.NewButton("Generate .env", func() {
		if createSampleCheck.Checked {
			// Create a sample schema
			samplePath := ".env.schema.json"
			if schemaPath.Text != "" {
				samplePath = schemaPath.Text
			}

			// Capture console output
			output := captureOutput(func() {
				err := generator.CreateSampleSchema(samplePath)
				if err != nil {
					fmt.Printf("Error creating sample schema: %v\n", err)
					return
				}
				fmt.Printf("✅ Created sample schema at %s\n", samplePath)
			})

			results.SetText(output)
			return
		}

		if schemaPath.Text == "" {
			dialog.ShowInformation("Error", "Please select a schema file", window)
			return
		}

		// Set environment variables as needed
		if outputPath.Text != "" {
			// In a real implementation, we would pass this to the generation function
		}

		// Capture console output
		output := captureOutput(func() {
			generator.GenerateEnv(schemaPath.Text)
		})

		results.SetText(output)
	})

	// Layout
	schemaSelectBox := container.NewHBox(schemaPath, browseSchemaBtn)
	outputSelectBox := container.NewHBox(outputPath, browseOutputBtn)

	return container.NewVBox(
		widget.NewLabel("Generate .env File from Schema"),
		createSampleCheck,
		widget.NewLabel("Schema file:"),
		schemaSelectBox,
		widget.NewLabel("Output file (optional):"),
		outputSelectBox,
		generateBtn,
		widget.NewLabel("Results:"),
		container.NewMax(results),
	)
}

// createAuditTab creates the tab for auditing .env files
func createAuditTab(window fyne.Window) fyne.CanvasObject {
	// File selection
	filePath := widget.NewEntry()
	filePath.SetPlaceHolder("Path to .env file")

	// Browse button
	browseBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if uri == nil {
				return
			}
			filePath.SetText(uri.URI().Path())
		}, window)
	})

	// Results display
	results := widget.NewMultiLineEntry()
	results.SetPlaceHolder("Audit results will appear here")
	results.Disable()

	// Audit button
	auditBtn := widget.NewButton("View Audit History", func() {
		if filePath.Text == "" {
			dialog.ShowInformation("Error", "Please select a file to audit", window)
			return
		}

		// Capture console output
		output := captureOutput(func() {
			audit.AuditEnv(filePath.Text)
		})

		results.SetText(output)
	})

	// Layout
	fileSelectBox := container.NewHBox(filePath, browseBtn)

	return container.NewVBox(
		widget.NewLabel("Audit .env File Changes"),
		widget.NewLabel("Select file to audit:"),
		fileSelectBox,
		auditBtn,
		widget.NewLabel("Results:"),
		container.NewMax(results),
	)
}

// createLintTab creates the tab for linting .env files
func createLintTab(window fyne.Window) fyne.CanvasObject {
	// File selection
	filePath := widget.NewEntry()
	filePath.SetPlaceHolder("Path to .env file")

	// Fix checkbox
	fixCheck := widget.NewCheck("Apply suggested fixes", nil)

	// Browse button
	browseBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if uri == nil {
				return
			}
			filePath.SetText(uri.URI().Path())
		}, window)
	})

	// Results display
	results := widget.NewMultiLineEntry()
	results.SetPlaceHolder("Linting results will appear here")
	results.Disable()

	// Lint button
	lintBtn := widget.NewButton("Lint", func() {
		if filePath.Text == "" {
			dialog.ShowInformation("Error", "Please select a file to lint", window)
			return
		}

		// Capture console output
		output := captureOutput(func() {
			linter.LintEnv(filePath.Text, fixCheck.Checked)
		})

		results.SetText(output)
	})

	// Layout
	fileSelectBox := container.NewHBox(filePath, browseBtn)

	return container.NewVBox(
		widget.NewLabel("Lint .env File"),
		widget.NewLabel("Select file to lint:"),
		fileSelectBox,
		fixCheck,
		lintBtn,
		widget.NewLabel("Results:"),
		container.NewMax(results),
	)
}

// createAboutTab creates the about tab
func createAboutTab() fyne.CanvasObject {
	logo := widget.NewLabel("EnvTool")
	logo.Alignment = fyne.TextAlignCenter
	logo.TextStyle = fyne.TextStyle{Bold: true}

	version := widget.NewLabel("Version 1.0.0")
	version.Alignment = fyne.TextAlignCenter

	description := widget.NewRichText(
		&widget.TextSegment{
			Text: "EnvTool is a utility for managing .env files with features for validation, comparison, encryption, synchronization, and more.\n\n" +
				"Developed with Go and compatible with multiple operating systems.",
			Style: widget.RichTextStyleDefault,
		},
	)

	features := widget.NewRichText(
		&widget.TextSegment{Text: "Core Features:", Style: widget.TextStyle{Bold: true}},
		&widget.TextSegment{Text: "\n• Validate .env file format and content"},
		&widget.TextSegment{Text: "\n• Compare and merge .env files"},
		&widget.TextSegment{Text: "\n• Encrypt and decrypt sensitive values"},
		&widget.TextSegment{Text: "\n• Synchronize with remote storage"},
		&widget.TextSegment{Text: "\n• Generate .env from templates"},
		&widget.TextSegment{Text: "\n• Track changes with audit history"},
		&widget.TextSegment{Text: "\n• Lint and fix naming conventions"},
	)

	return container.NewVBox(
		logo,
		version,
		container.NewCenter(
			container.NewVBox(
				description,
				widget.NewSeparator(),
				features,
			),
		),
	)
}

// captureOutput captures stdout during a function execution
func captureOutput(fn func()) string {
	// Create a pipe
	r, w, err := os.Pipe()
	if err != nil {
		log.Fatal(err)
	}

	// Save current stdout
	stdout := os.Stdout

	// Set stdout to our pipe
	os.Stdout = w

	// Run the function
	fn()

	// Close the writer side of the pipe to flush it
	w.Close()

	// Restore original stdout
	os.Stdout = stdout

	// Read captured output
	var buf strings.Builder
	if _, err := io.Copy(&buf, r); err != nil {
		log.Fatal(err)
	}

	return buf.String()
}
