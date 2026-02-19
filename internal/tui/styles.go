package tui

import "github.com/charmbracelet/lipgloss"

// ─── Theme definition ────────────────────────────────────────────────────────

// Theme holds every color token used by the TUI.
type Theme struct {
	Name string

	// Canvas
	Bg      lipgloss.Color
	Surface lipgloss.Color

	// Chrome
	Border   lipgloss.Color
	Accent   lipgloss.Color
	Selected lipgloss.Color

	// Semantic
	Text   lipgloss.Color
	Muted  lipgloss.Color
	Green  lipgloss.Color
	Red    lipgloss.Color
	Yellow lipgloss.Color
}

// ─── Built-in themes ─────────────────────────────────────────────────────────

var Themes = []Theme{
	{
		Name:     "dark",
		Bg:       lipgloss.Color("#0d1117"),
		Surface:  lipgloss.Color("#161b22"),
		Border:   lipgloss.Color("#30363d"),
		Accent:   lipgloss.Color("#58a6ff"),
		Selected: lipgloss.Color("#1f6feb"),
		Text:     lipgloss.Color("#e6edf3"),
		Muted:    lipgloss.Color("#8b949e"),
		Green:    lipgloss.Color("#3fb950"),
		Red:      lipgloss.Color("#f85149"),
		Yellow:   lipgloss.Color("#e3b341"),
	},
	{
		Name:     "light",
		Bg:       lipgloss.Color("#ffffff"),
		Surface:  lipgloss.Color("#f6f8fa"),
		Border:   lipgloss.Color("#d0d7de"),
		Accent:   lipgloss.Color("#0969da"),
		Selected: lipgloss.Color("#ddf4ff"),
		Text:     lipgloss.Color("#1f2328"),
		Muted:    lipgloss.Color("#656d76"),
		Green:    lipgloss.Color("#1a7f37"),
		Red:      lipgloss.Color("#cf222e"),
		Yellow:   lipgloss.Color("#9a6700"),
	},
	{
		Name:     "homebrew",
		Bg:       lipgloss.Color("#1a0a00"),
		Surface:  lipgloss.Color("#2a1500"),
		Border:   lipgloss.Color("#cc6600"),
		Accent:   lipgloss.Color("#ff8c00"),
		Selected: lipgloss.Color("#7a3300"),
		Text:     lipgloss.Color("#ffcc99"),
		Muted:    lipgloss.Color("#cc8844"),
		Green:    lipgloss.Color("#66cc44"),
		Red:      lipgloss.Color("#ff4444"),
		Yellow:   lipgloss.Color("#ffcc00"),
	},
	{
		Name:     "dracula",
		Bg:       lipgloss.Color("#282a36"),
		Surface:  lipgloss.Color("#1e1f29"),
		Border:   lipgloss.Color("#6272a4"),
		Accent:   lipgloss.Color("#bd93f9"),
		Selected: lipgloss.Color("#44475a"),
		Text:     lipgloss.Color("#f8f8f2"),
		Muted:    lipgloss.Color("#6272a4"),
		Green:    lipgloss.Color("#50fa7b"),
		Red:      lipgloss.Color("#ff5555"),
		Yellow:   lipgloss.Color("#f1fa8c"),
	},
	{
		Name:     "solarized",
		Bg:       lipgloss.Color("#002b36"),
		Surface:  lipgloss.Color("#073642"),
		Border:   lipgloss.Color("#586e75"),
		Accent:   lipgloss.Color("#268bd2"),
		Selected: lipgloss.Color("#094557"),
		Text:     lipgloss.Color("#839496"),
		Muted:    lipgloss.Color("#657b83"),
		Green:    lipgloss.Color("#859900"),
		Red:      lipgloss.Color("#dc322f"),
		Yellow:   lipgloss.Color("#b58900"),
	},
	{
		Name:     "nord",
		Bg:       lipgloss.Color("#2e3440"),
		Surface:  lipgloss.Color("#3b4252"),
		Border:   lipgloss.Color("#4c566a"),
		Accent:   lipgloss.Color("#88c0d0"),
		Selected: lipgloss.Color("#434c5e"),
		Text:     lipgloss.Color("#eceff4"),
		Muted:    lipgloss.Color("#d8dee9"),
		Green:    lipgloss.Color("#a3be8c"),
		Red:      lipgloss.Color("#bf616a"),
		Yellow:   lipgloss.Color("#ebcb8b"),
	},
	{
		Name:     "monokai",
		Bg:       lipgloss.Color("#272822"),
		Surface:  lipgloss.Color("#1e1f1c"),
		Border:   lipgloss.Color("#75715e"),
		Accent:   lipgloss.Color("#66d9e8"),
		Selected: lipgloss.Color("#49483e"),
		Text:     lipgloss.Color("#f8f8f2"),
		Muted:    lipgloss.Color("#75715e"),
		Green:    lipgloss.Color("#a6e22e"),
		Red:      lipgloss.Color("#f92672"),
		Yellow:   lipgloss.Color("#e6db74"),
	},
}

// ThemeByName returns a theme by name, defaulting to dark.
func ThemeByName(name string) Theme {
	for _, t := range Themes {
		if t.Name == name {
			return t
		}
	}
	return Themes[0]
}

// ─── Active style variables (set by setTheme) ─────────────────────────────────

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

	// Panel borders — Background set so lipgloss fills the inner canvas correctly.
	borderStyle       lipgloss.Style
	activeBorderStyle lipgloss.Style

	taskItemStyle         lipgloss.Style
	taskItemSelectedStyle lipgloss.Style
	taskCompletedStyle    lipgloss.Style

	detailHeaderStyle lipgloss.Style
	detailKeyStyle    lipgloss.Style
	detailValueStyle  lipgloss.Style

	providerActiveStyle   lipgloss.Style
	providerStyle         lipgloss.Style
	providerSelectedStyle lipgloss.Style

	statusBarStyle  lipgloss.Style
	statusOKStyle   lipgloss.Style
	statusErrStyle  lipgloss.Style
	statusLoadStyle lipgloss.Style

	keyStyle     lipgloss.Style
	keyDescStyle lipgloss.Style

	logStyle        lipgloss.Style
	logSuccessStyle lipgloss.Style
	logErrStyle     lipgloss.Style

	inputStyle   lipgloss.Style
	sectionStyle lipgloss.Style
)

// setTheme rebuilds every style variable from a Theme. Called whenever the
// theme changes. Every style that renders content inside a panel explicitly
// sets Background so no cell falls through to the terminal default.
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

	// The UnsetBackground() call on the inner content is intentional: we let
	// the border box's Background propagate. But lipgloss does NOT propagate
	// background from the outer box into child Render calls — each child must
	// set it explicitly. We do that everywhere in view.go; here we just make
	// sure the box itself has the right canvas color so any leftover padding
	// cells are also correct.
	borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Background(colorBg)

	activeBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Background(colorBg)

	// ── Task list ────────────────────────────────────────────────────────────
	taskItemStyle = lipgloss.NewStyle().
		Foreground(colorText).Background(colorBg).Padding(0, 1)

	taskItemSelectedStyle = lipgloss.NewStyle().
		Background(colorSelected).Foreground(colorText).Bold(true).Padding(0, 1)

	taskCompletedStyle = lipgloss.NewStyle().
		Foreground(colorMuted).Background(colorBg).Strikethrough(true).Padding(0, 1)

	// ── Detail panel ─────────────────────────────────────────────────────────
	detailHeaderStyle = lipgloss.NewStyle().
		Foreground(colorAccent).Background(colorBg).Bold(true)

	detailKeyStyle = lipgloss.NewStyle().
		Foreground(colorYellow).Background(colorBg).Bold(true)

	detailValueStyle = lipgloss.NewStyle().
		Foreground(colorText).Background(colorBg)

	// ── Provider / model panel ───────────────────────────────────────────────
	providerActiveStyle = lipgloss.NewStyle().
		Foreground(colorGreen).Background(colorBg).Bold(true)

	providerStyle = lipgloss.NewStyle().
		Foreground(colorText).Background(colorBg)

	providerSelectedStyle = lipgloss.NewStyle().
		Background(colorSelected).Foreground(colorText).Bold(true)

	// ── Status bar ───────────────────────────────────────────────────────────
	statusBarStyle = lipgloss.NewStyle().
		Background(colorSurface).Foreground(colorMuted).Padding(0, 1)

	statusOKStyle = lipgloss.NewStyle().
		Background(colorSurface).Foreground(colorGreen).Bold(true).Padding(0, 1)

	statusErrStyle = lipgloss.NewStyle().
		Background(colorSurface).Foreground(colorRed).Bold(true).Padding(0, 1)

	statusLoadStyle = lipgloss.NewStyle().
		Background(colorSurface).Foreground(colorYellow).Bold(true).Padding(0, 1)

	// ── Keybind bar ──────────────────────────────────────────────────────────
	keyStyle = lipgloss.NewStyle().
		Foreground(colorAccent).Background(colorSurface).Bold(true)

	keyDescStyle = lipgloss.NewStyle().
		Foreground(colorMuted).Background(colorSurface)

	// ── Log panel ────────────────────────────────────────────────────────────
	logStyle = lipgloss.NewStyle().
		Foreground(colorText).Background(colorBg).Padding(0, 1)

	logSuccessStyle = lipgloss.NewStyle().
		Foreground(colorGreen).Background(colorBg).Padding(0, 1)

	logErrStyle = lipgloss.NewStyle().
		Foreground(colorRed).Background(colorBg).Padding(0, 1)

	// ── Config screen ────────────────────────────────────────────────────────
	inputStyle = lipgloss.NewStyle().
		Foreground(colorYellow).Background(colorBg).Bold(true)

	sectionStyle = lipgloss.NewStyle().
		Foreground(colorMuted).Background(colorBg).Bold(true).Padding(0, 0, 0, 1)
}

func init() {
	setTheme(Themes[0]) // default: dark
}