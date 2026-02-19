package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds every color token used by the TUI.
type Theme struct {
	Name    string
	Bg      lipgloss.Color
	Surface lipgloss.Color
	Border  lipgloss.Color
	Accent  lipgloss.Color
	Selected lipgloss.Color
	Text    lipgloss.Color
	Muted   lipgloss.Color
	Green   lipgloss.Color
	Red     lipgloss.Color
	Yellow  lipgloss.Color
}

var Themes = []Theme{
	{
		Name: "dark", Bg: "#0d1117", Surface: "#161b22",
		Border: "#30363d", Accent: "#58a6ff", Selected: "#1f6feb",
		Text: "#e6edf3", Muted: "#8b949e", Green: "#3fb950", Red: "#f85149", Yellow: "#e3b341",
	},
	{
		Name: "light", Bg: "#ffffff", Surface: "#f6f8fa",
		Border: "#d0d7de", Accent: "#0969da", Selected: "#ddf4ff",
		Text: "#1f2328", Muted: "#656d76", Green: "#1a7f37", Red: "#cf222e", Yellow: "#9a6700",
	},
	{
		Name: "homebrew", Bg: "#1a0a00", Surface: "#2a1500",
		Border: "#cc6600", Accent: "#ff8c00", Selected: "#7a3300",
		Text: "#ffcc99", Muted: "#cc8844", Green: "#66cc44", Red: "#ff4444", Yellow: "#ffcc00",
	},
	{
		Name: "dracula", Bg: "#282a36", Surface: "#1e1f29",
		Border: "#6272a4", Accent: "#bd93f9", Selected: "#44475a",
		Text: "#f8f8f2", Muted: "#6272a4", Green: "#50fa7b", Red: "#ff5555", Yellow: "#f1fa8c",
	},
	{
		Name: "solarized", Bg: "#002b36", Surface: "#073642",
		Border: "#586e75", Accent: "#268bd2", Selected: "#094557",
		Text: "#839496", Muted: "#657b83", Green: "#859900", Red: "#dc322f", Yellow: "#b58900",
	},
	{
		Name: "nord", Bg: "#2e3440", Surface: "#3b4252",
		Border: "#4c566a", Accent: "#88c0d0", Selected: "#434c5e",
		Text: "#eceff4", Muted: "#d8dee9", Green: "#a3be8c", Red: "#bf616a", Yellow: "#ebcb8b",
	},
	{
		Name: "monokai", Bg: "#272822", Surface: "#1e1f1c",
		Border: "#75715e", Accent: "#66d9e8", Selected: "#49483e",
		Text: "#f8f8f2", Muted: "#75715e", Green: "#a6e22e", Red: "#f92672", Yellow: "#e6db74",
	},
}

func ThemeByName(name string) Theme {
	for _, t := range Themes {
		if t.Name == name {
			return t
		}
	}
	return Themes[0]
}

// ─── Active color variables ───────────────────────────────────────────────────

var (
	colorBg       lipgloss.Color
	colorSurface  lipgloss.Color
	colorBorder   lipgloss.Color
	colorAccent   lipgloss.Color
	colorYellow   lipgloss.Color
	colorGreen    lipgloss.Color
	colorRed      lipgloss.Color
	colorMuted    lipgloss.Color
	colorText     lipgloss.Color
	colorSelected lipgloss.Color

	// borderStyle and activeBorderStyle are the ONLY two pre-built styles kept
	// at package level. Everything else is built inline in view.go to ensure
	// Background is always set correctly. These two exist so the box container
	// itself paints its background on any padding cells lipgloss adds.
	borderStyle       lipgloss.Style
	activeBorderStyle lipgloss.Style
)

// setTheme rebuilds all package-level style variables. Called on startup and
// whenever the theme changes.
func setTheme(t Theme) {
	colorBg       = t.Bg
	colorSurface  = t.Surface
	colorBorder   = t.Border
	colorAccent   = t.Accent
	colorYellow   = t.Yellow
	colorGreen    = t.Green
	colorRed      = t.Red
	colorMuted    = t.Muted
	colorText     = t.Text
	colorSelected = t.Selected

	borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Background(colorBg)

	activeBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Background(colorBg)
}

func init() { setTheme(Themes[0]) }