package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Base palette
	colorBg      = lipgloss.Color("#0d1117")
	colorSurface = lipgloss.Color("#161b22")
	colorBorder  = lipgloss.Color("#30363d")
	colorAccent  = lipgloss.Color("#58a6ff")
	colorYellow  = lipgloss.Color("#e3b341")
	colorGreen   = lipgloss.Color("#3fb950")
	colorRed     = lipgloss.Color("#f85149")
	colorMuted   = lipgloss.Color("#8b949e")
	colorText    = lipgloss.Color("#e6edf3")
	colorSelected = lipgloss.Color("#1f6feb")

	// Panel borders
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent)

	// Task list items
	taskItemStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	taskItemSelectedStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(colorText).
				Bold(true).
				Padding(0, 1)

	taskCompletedStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Strikethrough(true).
				Padding(0, 1)

	// Detail panel
	detailHeaderStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	detailKeyStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(colorText)

	// Model panel
	providerActiveStyle = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	providerStyle = lipgloss.NewStyle().
			Foreground(colorText)

	providerSelectedStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(colorText).
				Bold(true)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorMuted).
			Padding(0, 1)

	statusOKStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorGreen).
			Bold(true).
			Padding(0, 1)

	statusErrStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorRed).
			Bold(true).
			Padding(0, 1)

	statusLoadStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorYellow).
			Bold(true).
			Padding(0, 1)

	// Keybind bar
	keyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	keyDescStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Execution log
	logStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	logSuccessStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Padding(0, 1)

	logErrStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Padding(0, 1)

	// Input style
	inputStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	// Panel section header
	sectionStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(true).
			Padding(0, 0, 0, 1)
)
