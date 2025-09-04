package ui

import "github.com/charmbracelet/lipgloss"

// Theme colors
var (
	// Primary interface color - easily modifiable
	PrimaryColor = "#00ADD8" // Official Go logo blue color

	// Secondary colors
	SecondaryColor = "240" // Gray
	ErrorColor     = "1"   // Red
	SuccessColor   = "36"  // Green (for reference if needed)
)

// Styles struct centralizes all lipgloss styles
type Styles struct {
	// Layout
	App    lipgloss.Style
	Header lipgloss.Style

	// Search styles
	SearchFocused   lipgloss.Style
	SearchUnfocused lipgloss.Style

	// Table styles
	TableFocused   lipgloss.Style
	TableUnfocused lipgloss.Style
	Selected       lipgloss.Style

	// Info and help styles
	SortInfo lipgloss.Style
	HelpText lipgloss.Style

	// Error and confirmation styles
	Error lipgloss.Style

	// Form styles (for add/edit forms)
	FormTitle     lipgloss.Style
	FormField     lipgloss.Style
	FormHelp      lipgloss.Style
	FormContainer lipgloss.Style
	Label         lipgloss.Style
	FocusedLabel  lipgloss.Style
}

// NewStyles creates a new Styles struct with the given terminal width
func NewStyles(width int) Styles {
	return Styles{
		// Main app container
		App: lipgloss.NewStyle().
			Padding(1),

		// Header style
		Header: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryColor)).
			Bold(true).
			Align(lipgloss.Center),

		// Search styles
		SearchFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(PrimaryColor)).
			Padding(0, 1),

		SearchUnfocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(SecondaryColor)).
			Padding(0, 1),

		// Table styles
		TableFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(PrimaryColor)),

		TableUnfocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(SecondaryColor)),

		// Style for selected items
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color(PrimaryColor)).
			Bold(false),

		// Info styles
		SortInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SecondaryColor)),

		HelpText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SecondaryColor)).
			MarginTop(1),

		// Error style
		Error: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ErrorColor)).
			Padding(1, 2),

		// Form styles
		FormTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color(PrimaryColor)).
			Padding(0, 1),

		FormField: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryColor)),

		FormHelp: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")),

		FormContainer: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(PrimaryColor)).
			Padding(1, 2),

		Label: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SecondaryColor)),

		FocusedLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryColor)).
			Bold(true),
	}
}

// Application ASCII title
const asciiTitle = `
 _____ _____ __ __ _____
|   __|   __|  |  |     |
|__   |__   |     | | | |
|_____|_____|__|__|_|_|_|
`
