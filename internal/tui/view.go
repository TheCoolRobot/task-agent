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


func (m Model) View() string {
	if m.width == 0 {
		return "Initializing…"
	}
	if m.width < 60 || m.height < 10 {
		return lipgloss.NewStyle().Padding(1, 2).Render("Terminal too small — please resize!")
	}

	// Full-screen modes
	if m.searching {
		return m.viewSearchOverlay()
	}
	if m.activePane == paneConfig {
		return m.viewConfigScreen()
	}

	// Main 3-panel layout
	leftW  := m.width * 42 / 100
	rightW := m.width - leftW - 1
	bodyH  := m.height - 3 // header(1) + statusbar(1) + keybinds(1)

	detailH := bodyH * 60 / 100
	modelH  := bodyH - detailH

	left   := m.viewTaskList(leftW, bodyH)
	detail := m.viewDetail(rightW, detailH)
	model  := m.viewModelPane(rightW, modelH)

	right := detail
	if m.activePane == paneLog {
		right = m.viewLogPanel(rightW, bodyH)
	} else {
		right = lipgloss.JoinVertical(lipgloss.Left, detail, model)
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return lipgloss.JoinVertical(lipgloss.Left,
		m.viewHeader(),
		body,
		m.viewStatusBar(),
		m.viewKeybinds(),
	)
}

// ─── Header ──────────────────────────────────────────────────────────────────

func (m Model) viewHeader() string {
	logo := lipgloss.NewStyle().
		Foreground(colorAccent).Bold(true).
		Render("⚡ task-agent")

	badge := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(fmt.Sprintf("  %s / %s", m.modelPane.activeProvider, m.modelPane.activeModel))

	spin := ""
	if m.loading || m.executing {
		spin = "  " + m.spinner.View()
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Background(colorBg).
		BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorBorder).
		Render(logo + badge + spin)
}

// ─── Task List Panel ─────────────────────────────────────────────────────────

func (m Model) viewTaskList(w, h int) string {
	active := m.activePane == paneTasks
	inner  := w - 2
	visH   := h - 3

	var rows []string
	header := sectionStyle.Render("Tasks")
	if len(m.filteredTasks) > 0 {
		header += keyDescStyle.Render(fmt.Sprintf(" (%d)", len(m.filteredTasks)))
	}
	rows = append(rows, header)

	if m.loading && len(m.tasks) == 0 {
		rows = append(rows, keyDescStyle.Render("  "+m.spinner.View()+" Loading…"))
	} else if len(m.filteredTasks) == 0 {
		rows = append(rows, keyDescStyle.Render("  No tasks — press r to refresh"))
	} else {
		for i, task := range m.filteredTasks {
			if i < m.taskScroll || i >= m.taskScroll+visH {
				continue
			}
			rows = append(rows, m.renderTaskRow(task, i == m.taskCursor, inner))
		}
	}

	// Scroll indicator
	total := len(m.filteredTasks)
	if total > visH {
		pct := 100 * m.taskScroll / (total - visH + 1)
		rows = append(rows, keyDescStyle.Render(fmt.Sprintf("  ↕ %d%%", pct)))
	}

	box := borderStyle
	if active {
		box = activeBorderStyle
	}
	return box.Width(w).Height(h).Render(strings.Join(rows, "\n"))
}

func (m Model) renderTaskRow(task asana.Task, selected bool, maxW int) string {
	status := task.StatusIcon()
	pri    := task.PriorityIcon()
	name   := task.Name
	if max := maxW - 9; len([]rune(name)) > max && max > 3 {
		name = string([]rune(name)[:max-1]) + "…"
	}
	line := fmt.Sprintf(" %s %s %s", status, pri, name)
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
	box := borderStyle

	var rows []string
	rows = append(rows, sectionStyle.Render("Task Details"))

	if len(m.filteredTasks) == 0 || m.taskCursor >= len(m.filteredTasks) {
		rows = append(rows, keyDescStyle.Render("  Select a task"))
	} else {
		rows = append(rows, m.renderDetail(m.filteredTasks[m.taskCursor], w-4)...)
	}

	return box.Width(w).Height(h).Render(strings.Join(rows, "\n"))
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
	status := "⏳ Incomplete"
	if task.Completed {
		status = "✅ Complete"
	}
	kv("Status", status)
	kv("Priority", task.Priority)
	kv("Due", task.DueDate)
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
		// word-wrap
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

// ─── Model/Provider Switcher Panel ───────────────────────────────────────────

func (m Model) viewModelPane(w, h int) string {
	active := m.activePane == paneModel
	box    := borderStyle
	if active {
		box = activeBorderStyle
	}

	sub := "Providers"
	if m.modelSubPane == 1 {
		sub = "Models"
	}
	title := sectionStyle.Render(fmt.Sprintf("Model › %s", sub))
	hint  := keyDescStyle.Render("  [Tab] toggle · [Enter] select")

	var rows []string
	rows = append(rows, title+hint)

	if m.modelSubPane == 0 {
		for i, prov := range m.modelPane.providers {
			isSel     := i == m.modelPane.providerCursor && active
			isCurrent := prov.ID == m.modelPane.activeProvider
			marker := "  "
			if isCurrent {
				marker = "▶ "
			}
			label := marker + prov.Name
			switch {
			case isSel:
				rows = append(rows, providerSelectedStyle.Width(w-4).Render(label))
			case isCurrent:
				rows = append(rows, providerActiveStyle.Render(label))
			default:
				rows = append(rows, providerStyle.Render(label))
			}
		}
	} else {
		prov := m.modelPane.providers[m.modelPane.providerCursor]
		rows = append(rows, keyDescStyle.Render("  "+prov.Name))
		for i, mod := range prov.Models {
			isSel     := i == m.modelPane.modelCursor && active
			isCurrent := mod == m.modelPane.activeModel && prov.ID == m.modelPane.activeProvider
			marker := "  "
			if isCurrent {
				marker = "▶ "
			}
			label := marker + mod
			switch {
			case isSel:
				rows = append(rows, providerSelectedStyle.Width(w-4).Render(label))
			case isCurrent:
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
	box := activeBorderStyle

	visH := h - 3
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

	return box.Width(w).Height(h).Render(strings.Join(rows, "\n"))
}

// ─── Config Screen (full-screen) ─────────────────────────────────────────────

func (m Model) viewConfigScreen() string {
	w, h := m.width, m.height

	// Centered card
	cardW := min(w-4, 72)
	cardH := h - 4

	title := lipgloss.NewStyle().
		Foreground(colorAccent).Bold(true).
		Render("🔧  Configuration")

	hint := keyDescStyle.Render("Tab/Shift-Tab navigate · Enter save field · Ctrl-S save & close · Esc cancel")

	var rows []string
	rows = append(rows, title)
	rows = append(rows, hint)
	rows = append(rows, "")

	for i, f := range configFields {
		isFocused := i == m.cfgCursor
		label := f.label
		if isFocused {
			label = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("› "+label)
		} else {
			label = keyDescStyle.Render("  "+label)
		}

		var valueStr string
		if len(f.options) > 0 {
			// Horizontal picker
			cursor := m.cfgOptCursors[i]
			var opts []string
			for j, opt := range f.options {
				if j == cursor {
					opts = append(opts, lipgloss.NewStyle().
						Foreground(colorYellow).Bold(true).
						Render("[ "+opt+" ]"))
				} else {
					opts = append(opts, keyDescStyle.Render("  "+opt+"  "))
				}
			}
			valueStr = strings.Join(opts, " ")
		} else {
			// Styled text input
			inp := m.cfgInputs[i]
			if isFocused {
				valueStr = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(colorAccent).
					Width(cardW - 6).
					Render(inp.View())
			} else {
				display := inp.Value()
				if f.secret && display != "" {
					display = strings.Repeat("•", len(display))
					if len(display) > 32 {
						display = display[:32]
					}
				}
				if display == "" {
					display = keyDescStyle.Render("(not set)")
				}
				valueStr = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(colorBorder).
					Width(cardW - 6).
					Foreground(colorText).
					Render(display)
			}
		}

		rows = append(rows, label)
		rows = append(rows, valueStr)
		rows = append(rows, "")
	}

	// Provider models note
	for i, f := range configFields {
		if f.key == "provider" {
			chosen := f.options[m.cfgOptCursors[i]]
			if prov, ok := ai.GetProvider(chosen); ok {
				rows = append(rows, keyDescStyle.Render(
					fmt.Sprintf("  Available models for %s: %s", prov.Name, strings.Join(prov.Models, ", ")),
				))
			}
			break
		}
	}

	card := activeBorderStyle.Width(cardW).Height(cardH).Render(strings.Join(rows, "\n"))

	// Center horizontally
	pad := (w - cardW) / 2
	if pad < 0 {
		pad = 0
	}
	return lipgloss.NewStyle().PaddingLeft(pad).PaddingTop(1).Render(card)
}

// ─── Search Overlay ──────────────────────────────────────────────────────────

func (m Model) viewSearchOverlay() string {
	box := lipgloss.NewStyle().
		Width(m.width - 4).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(0, 1)

	content := inputStyle.Render("/ ") + m.searchInput.View() +
		keyDescStyle.Render("   Enter=search · Esc=cancel")

	return lipgloss.NewStyle().
		Width(m.width).Height(m.height).
		Padding(m.height/2-1, 2).
		Render(box.Render(content))
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
	if len(msg) > m.width-2 {
		msg = msg[:m.width-5] + "…"
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
			{"Tab", "next field"},
			{"←→/Enter", "pick option"},
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
