//go:build !windows

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
)

// fyneUIStarter implements the UI startup logic for non-Windows platforms
func fyneUIStarter() error {
	// Initialize Fyne application
	a := app.New()
	w := a.NewWindow("EnvTool - Environment Management")
	w.Resize(fyne.NewSize(800, 600))

	// Create tabs for different functionalities
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Validate", nil, createValidateTab(w)),
		container.NewTabItemWithIcon("Compare", nil, createCompareTab(w)),
		container.NewTabItemWithIcon("Encrypt", nil, createEncryptTab(w)),
		container.NewTabItemWithIcon("Sync", nil, createSyncTab(w)),
		container.NewTabItemWithIcon("Generate", nil, createGenerateTab(w)),
		container.NewTabItemWithIcon("Audit", nil, createAuditTab(w)),
		container.NewTabItemWithIcon("Lint", nil, createLintTab(w)),
		container.NewTabItemWithIcon("About", nil, createAboutTab()),
	)

	tabs.SetTabLocation(container.TabLocationLeading)

	// Set window content and show
	w.SetContent(tabs)
	w.ShowAndRun()

	return nil
}
