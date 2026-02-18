package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thecoolrobot/task-agent/internal/ai"
	"github.com/thecoolrobot/task-agent/internal/asana"
	"github.com/thecoolrobot/task-agent/internal/config"
)

// Row budget constants — every pixel accounted for.
const (
	headerRows   = 2 // text line + bottom border
	statusRows   = 1
	keybindRows  = 1
	fixedRows    = headerRows + statusRows + keybindRows // = 4
	borderRows   = 2 // top + bottom border on each panel
)

// ─── Top-level View ──────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing…"
	}
	if m.width < 60 || m.height < 10 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			"Terminal too small — please resize!")
	}
	if m.searching {
		return m.viewSearchOverlay()
	}
	if m.activePane == paneConfig {
		return m.viewConfigScreen()
	}

	// bodyH is the exact number of terminal rows the panels can occupy,
	// including their own borders.
	bodyH  := m.height - fixedRows
	leftW  := m.width * 42 / 100
	rightW := m.width - leftW

	// Right column splits into detail + model; both borders included in budget.
	detailH := bodyH * 60 / 100
	if detailH < borderRows+1 {
		detailH = borderRows + 1
	}
	modelH := bodyH - detailH
	if modelH < borderRows+1 {
		modelH = borderRows + 1
	}

	left := m.viewTaskList(leftW, bodyH)

	var right string
	if m.activePane == paneLog {
		right = m.viewLogPanel(rightW, bodyH)
	} else {
		right = lipgloss.JoinVertical(lipgloss.Left,
			m.viewDetail(rightW, detailH),
			m.viewModelPane(rightW, modelH),
		)
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	// Clamp body to exact bodyH rows to prevent overflow
	bodyLines := strings.Split(body, "\n")
	if len(bodyLines) > bodyH {
		bodyLines = bodyLines[:bodyH]
	}
	body = strings.Join(bodyLines, "\n")

	return strings.Join([]string{
		m.viewHeader(),
		body,
		m.viewStatusBar(),
		m.viewKeybinds(),
	}, "\n")
}

// ─── Header ──────────────────────────────────────────────────────────────────

func (m Model) viewHeader() string {
	logo  := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("⚡ task-agent")
	badge := lipgloss.NewStyle().Foreground(colorMuted).
		Render(fmt.Sprintf("  %s / %s", m.modelPane.activeProvider, m.modelPane.activeModel))
	spin := ""
	if m.loading || m.executing {
		spin = "  " + m.spinner.View()
	}
	// BorderBottom = 1 extra row → total headerRows = 2
	return lipgloss.NewStyle().
		Width(m.width).
		//Background(colorBg).
		BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorBorder).
		Render(logo + badge + spin)
}

// ─── Task List Panel ─────────────────────────────────────────────────────────

func (m Model) viewTaskList(w, h int) string {
	active := m.activePane == paneTasks
	box    := borderStyle
	if active {
		box = activeBorderStyle
	}

	// Content rows = panel height minus the 2 border rows minus 1 title row
	innerH := h - borderRows - 1
	if innerH < 1 {
		innerH = 1
	}

	var rows []string
	header := sectionStyle.Render("Tasks")
	if len(m.filteredTasks) > 0 {
		header += keyDescStyle.Render(fmt.Sprintf(" (%d)", len(m.filteredTasks)))
	}
	rows = append(rows, header)

	switch {
	case m.loading && len(m.tasks) == 0:
		rows = append(rows, keyDescStyle.Render("  "+m.spinner.View()+" Loading…"))
	case len(m.filteredTasks) == 0:
		rows = append(rows, keyDescStyle.Render("  No tasks — press r to refresh"))
	default:
		shown := 0
		for i, task := range m.filteredTasks {
			if i < m.taskScroll {
				continue
			}
			if shown >= innerH {
				break
			}
			rows = append(rows, m.renderTaskRow(task, i == m.taskCursor, w-2))
			shown++
		}
		total := len(m.filteredTasks)
		if total > innerH {
			pct := 0
			if denom := total - innerH; denom > 0 {
				pct = 100 * m.taskScroll / denom
			}
			rows = append(rows, keyDescStyle.Render(fmt.Sprintf("  ↕ %d%%", pct)))
		}
	}

	return box.Width(w).Height(h).Render(strings.Join(rows, "\n"))
}

func (m Model) renderTaskRow(task asana.Task, selected bool, maxW int) string {
	name := task.Name
	if max := maxW - 9; len([]rune(name)) > max && max > 3 {
		name = string([]rune(name)[:max-1]) + "…"
	}
	line := fmt.Sprintf(" %s %s %s", task.StatusIcon(), task.PriorityIcon(), name)
	switch {
	case selected:
		return taskItemSelectedStyle.Width(maxW).Render(line)
	case task.Completed:
		return taskCompletedStyle.Width(maxW).Render(line)
	default:
		return taskItemStyle.Width(maxW).Render(line)
	}
}

// ─── Task Detail Panel ───────────────────────────────────────────────────────

func (m Model) viewDetail(w, h int) string {
	var rows []string
	rows = append(rows, sectionStyle.Render("Task Details"))

	if len(m.filteredTasks) == 0 || m.taskCursor >= len(m.filteredTasks) {
		rows = append(rows, keyDescStyle.Render("  Select a task"))
	} else {
		rows = append(rows, m.renderDetail(m.filteredTasks[m.taskCursor], w-4)...)
	}

	return borderStyle.Width(w).Height(h).Render(strings.Join(rows, "\n"))
}

func (m Model) renderDetail(task asana.Task, maxW int) []string {
	var rows []string
	rows = append(rows, detailHeaderStyle.Width(maxW).Render(task.Name))
	rows = append(rows, "")

	kv := func(k, v string) {
		if v == "" {
			return
		}
		rows = append(rows, detailKeyStyle.Render(k+": ")+detailValueStyle.Render(v))
	}

	kv("ID", task.GetID())
	if task.Completed {
		kv("Status", "✅ Complete")
	} else {
		kv("Status", "⏳ Incomplete")
	}
	kv("Priority", task.Priority)
	kv("Due",      task.DueDate)
	kv("Assignee", task.Assignee.Name)

	if len(task.Tags) > 0 {
		names := make([]string, len(task.Tags))
		for i, t := range task.Tags {
			names[i] = t.Name
		}
		kv("Tags", strings.Join(names, ", "))
	}

	if task.Notes != "" {
		rows = append(rows, "")
		rows = append(rows, detailKeyStyle.Render("Description:"))
		words := strings.Fields(task.Notes)
		line := ""
		for _, w := range words {
			if len(line)+len(w)+1 > maxW-2 {
				rows = append(rows, detailValueStyle.Render("  "+line))
				line = w
			} else {
				if line != "" {
					line += " "
				}
				line += w
			}
		}
		if line != "" {
			rows = append(rows, detailValueStyle.Render("  "+line))
		}
	}
	return rows
}

// ─── Model/Provider Panel ────────────────────────────────────────────────────

func (m Model) viewModelPane(w, h int) string {
	active := m.activePane == paneModel
	box    := borderStyle
	if active {
		box = activeBorderStyle
	}

	sub := "Providers  [Tab→models]"
	if m.modelSubPane == 1 {
		sub = "Models  [Tab→providers]"
	}

	var rows []string
	rows = append(rows, sectionStyle.Render("Model › "+sub))

	if m.modelSubPane == 0 {
		for i, prov := range m.modelPane.providers {
			marker := "  "
			if prov.ID == m.modelPane.activeProvider {
				marker = "▶ "
			}
			label := marker + prov.Name
			switch {
			case i == m.modelPane.providerCursor && active:
				rows = append(rows, providerSelectedStyle.Width(w-4).Render(label))
			case prov.ID == m.modelPane.activeProvider:
				rows = append(rows, providerActiveStyle.Render(label))
			default:
				rows = append(rows, providerStyle.Render(label))
			}
		}
	} else {
		prov := m.modelPane.providers[m.modelPane.providerCursor]
		rows = append(rows, keyDescStyle.Render("  "+prov.Name))
		for i, mod := range prov.Models {
			marker := "  "
			if mod == m.modelPane.activeModel && prov.ID == m.modelPane.activeProvider {
				marker = "▶ "
			}
			label := marker + mod
			switch {
			case i == m.modelPane.modelCursor && active:
				rows = append(rows, providerSelectedStyle.Width(w-4).Render(label))
			case mod == m.modelPane.activeModel && prov.ID == m.modelPane.activeProvider:
				rows = append(rows, providerActiveStyle.Render(label))
			default:
				rows = append(rows, providerStyle.Render(label))
			}
		}
	}

	return box.Width(w).Height(h).Render(strings.Join(rows, "\n"))
}

// ─── Execution Log Panel ─────────────────────────────────────────────────────

func (m Model) viewLogPanel(w, h int) string {
	visH := h - borderRows - 1
	if visH < 1 {
		visH = 1
	}

	start := 0
	if len(m.logLines) > visH {
		start = len(m.logLines) - visH
	}

	var rows []string
	rows = append(rows, sectionStyle.Render("Execution Log"))

	for _, line := range m.logLines[start:] {
		var s string
		switch line.kind {
		case "ok":
			s = logSuccessStyle.Render(line.text)
		case "err":
			s = logErrStyle.Render(line.text)
		case "dim":
			s = keyDescStyle.Render(line.text)
		default:
			s = logStyle.Render(line.text)
		}
		rows = append(rows, s)
	}

	if m.executing {
		rows = append(rows, logStyle.Render(m.spinner.View()+" Working…"))
	}

	return activeBorderStyle.Width(w).Height(h).Render(strings.Join(rows, "\n"))
}

// ─── Config Screen ───────────────────────────────────────────────────────────

func (m Model) viewConfigScreen() string {
	// Card is always smaller than the terminal so Place() can center it
	cardW := min(m.width-4, 72)
	// Each field = label(1) + input(3 with border) + gap(1) = 5 rows
	// Header = title(1) + hint(1) + gap(1) = 3 rows
	// Footer = scroll hint(1) + models note(1) = 2 rows
	maxFieldsVisible := (m.height - 4 - 3 - 2) / 5
	if maxFieldsVisible < 1 {
		maxFieldsVisible = 1
	}
	cardH := 3 + (maxFieldsVisible * 5) + 3
	if cardH > m.height-2 {
		cardH = m.height - 2
	}

	title := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("🔧  Configuration")
	hint  := keyDescStyle.Render("Tab/↑↓ navigate · ←→ options · Ctrl-S save · Esc cancel")

	var rows []string
	rows = append(rows, title)
	rows = append(rows, hint)
	rows = append(rows, "")

	// Scroll window: keep focused field in view
	fieldStart := 0
	if m.cfgCursor >= maxFieldsVisible {
		fieldStart = m.cfgCursor - maxFieldsVisible + 1
	}
	fieldEnd := fieldStart + maxFieldsVisible
	if fieldEnd > len(configFields) {
		fieldEnd = len(configFields)
	}

	for i := fieldStart; i < fieldEnd; i++ {
		f         := configFields[i]
		isFocused := i == m.cfgCursor
		inputW    := cardW - 8

		var label string
		if isFocused {
			label = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("› " + f.label)
		} else {
			label = keyDescStyle.Render("  " + f.label)
		}

		var valueStr string
		if len(f.options) > 0 {
			cursor := m.cfgOptCursors[i]
			var opts []string
			for j, opt := range f.options {
				if j == cursor {
					opts = append(opts, lipgloss.NewStyle().
						Foreground(colorYellow).Bold(true).Render("[ "+opt+" ]"))
				} else {
					opts = append(opts, keyDescStyle.Render("  "+opt+"  "))
				}
			}
			valueStr = strings.Join(opts, " ")
		} else {
			inp := m.cfgInputs[i]
			if isFocused {
				valueStr = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent).
					Width(inputW).Render(inp.View())
			} else {
				display := inp.Value()
				if f.secret && display != "" {
					display = strings.Repeat("•", min(len(display), 32))
				}
				if display == "" {
					display = "(not set)"
				}
				valueStr = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder).
					Foreground(colorMuted).Width(inputW).Render(display)
			}
		}

		rows = append(rows, label)
		rows = append(rows, valueStr)
		rows = append(rows, "")
	}

	// Scroll position hint
	if len(configFields) > maxFieldsVisible {
		rows = append(rows, keyDescStyle.Render(
			fmt.Sprintf("  field %d / %d", m.cfgCursor+1, len(configFields))))
	}

	// Available models for selected provider
	for i, f := range configFields {
		if f.key == "provider" {
			chosen := f.options[m.cfgOptCursors[i]]
			if prov, ok := ai.GetProvider(chosen); ok {
				rows = append(rows, keyDescStyle.Render(
					"  Models: "+strings.Join(prov.Models, ", ")))
			}
			break
		}
	}

	card := activeBorderStyle.Width(cardW).Height(cardH).Render(strings.Join(rows, "\n"))

	// Place centers the card within the full terminal dimensions
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
}

// ─── Search Overlay ──────────────────────────────────────────────────────────

func (m Model) viewSearchOverlay() string {
	boxW := min(m.width-4, 64)
	content := inputStyle.Render("/ ") + m.searchInput.View() +
		keyDescStyle.Render("   Enter=search · Esc=cancel")
	box := lipgloss.NewStyle().
		Width(boxW).
		Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent).
		Padding(0, 1).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// ─── Status Bar ──────────────────────────────────────────────────────────────

func (m Model) viewStatusBar() string {
	var s lipgloss.Style
	switch m.statusKind {
	case "ok":
		s = statusOKStyle
	case "err":
		s = statusErrStyle
	case "loading":
		s = statusLoadStyle
	default:
		s = statusBarStyle
	}
	msg := m.statusMsg
	runes := []rune(msg)
	if len(runes) > m.width-2 {
		msg = string(runes[:m.width-5]) + "…"
	}
	return s.Width(m.width).Render(msg)
}

// ─── Keybind Bar ─────────────────────────────────────────────────────────────

func (m Model) viewKeybinds() string {
	binds := []struct{ k, d string }{
		{"↑↓/jk", "nav"},
		{"Enter", "execute"},
		{"Tab", "pane"},
		{"/", "search"},
		{"C", "config"},
		{"L", "log"},
		{"r", "refresh"},
		{"q", "quit"},
	}
	if m.activePane == paneConfig {
		binds = []struct{ k, d string }{
			{"Tab/↑↓", "navigate"},
			{"←→", "pick option"},
			{"Ctrl-S", "save & close"},
			{"Esc", "cancel"},
		}
	}
	var parts []string
	for _, b := range binds {
		parts = append(parts, keyStyle.Render(b.k)+keyDescStyle.Render(" "+b.d))
	}
	return lipgloss.NewStyle().
		Width(m.width).Background(colorSurface).
		Render(" " + strings.Join(parts, keyDescStyle.Render(" · ")))
}

// ─── Run ─────────────────────────────────────────────────────────────────────

// Run starts the Bubble Tea program.
func Run(cfg *config.Config, client *asana.Client) error {
	m := New(cfg, client)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}