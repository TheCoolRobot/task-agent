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

const (
	headerRows  = 2 // title line + bottom border
	statusRows  = 1
	keybindRows = 1
	fixedRows   = headerRows + statusRows + keybindRows
)

// panelInnerW returns usable character width inside a rounded-border panel.
func panelInnerW(outerW int) int {
	if outerW-2 < 1 {
		return 1
	}
	return outerW - 2
}

// panelInnerH returns usable line count inside a rounded-border panel.
func panelInnerH(outerH int) int {
	if outerH-2 < 1 {
		return 1
	}
	return outerH - 2
}

// padLines extends lines to exactly n entries using blank bg-colored rows.
func padLines(lines []string, n, iW int) []string {
	blank := lipgloss.NewStyle().Background(colorBg).Width(iW).Render("")
	for len(lines) < n {
		lines = append(lines, blank)
	}
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

// ─── Top-level View ──────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
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

	bodyH := m.height - fixedRows

	// Two-column layout. leftW + rightW must equal m.width exactly.
	leftW  := m.width * 40 / 100
	rightW := m.width - leftW

	// Right column stacks two panels vertically.
	detailH := bodyH * 60 / 100
	if detailH < 4 {
		detailH = 4
	}
	modelH := bodyH - detailH
	if modelH < 4 {
		modelH = 4
	}

	leftPanel := m.viewTaskList(leftW, bodyH)

	var rightPanel string
	if m.activePane == paneLog {
		rightPanel = m.viewLogPanel(rightW, bodyH)
	} else {
		rightPanel = lipgloss.JoinVertical(lipgloss.Left,
			m.viewDetail(rightW, detailH),
			m.viewModelPane(rightW, modelH),
		)
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	return lipgloss.JoinVertical(lipgloss.Left,
		m.viewHeader(),
		body,
		m.viewStatusBar(),
		m.viewKeybinds(),
	)
}

// ─── Header ──────────────────────────────────────────────────────────────────

func (m Model) viewHeader() string {
	logo  := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Background(colorBg).Render("task-agent")
	badge := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).
		Render("  " + m.modelPane.activeProvider + " / " + m.modelPane.activeModel)
	spin := ""
	if m.loading || m.executing {
		spin = "  " + m.spinner.View()
	}
	return lipgloss.NewStyle().
		Width(m.width).Background(colorBg).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorBorder).
		Render(logo + badge + spin)
}

// ─── Task List Panel ─────────────────────────────────────────────────────────

func (m Model) viewTaskList(outerW, outerH int) string {
	iW := panelInnerW(outerW)
	iH := panelInnerH(outerH)

	// Helper styles (inline so they always use current theme colors)
	titleSt := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).Bold(true).Width(iW)
	mutedSt := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).Width(iW)

	var lines []string

	title := "Tasks"
	if len(m.filteredTasks) > 0 {
		title = fmt.Sprintf("Tasks (%d)", len(m.filteredTasks))
	}
	lines = append(lines, titleSt.Render(title))

	contentH := iH - 1 // subtract title row

	switch {
	case m.loading && len(m.tasks) == 0:
		lines = append(lines, mutedSt.Render(" "+m.spinner.View()+" Loading..."))

	case len(m.filteredTasks) == 0:
		lines = append(lines, mutedSt.Render(" No tasks — press r to refresh"))

	default:
		shown := 0
		for i, task := range m.filteredTasks {
			if i < m.taskScroll {
				continue
			}
			if shown >= contentH {
				break
			}
			lines = append(lines, m.renderTaskRow(task, i == m.taskCursor, iW))
			shown++
		}
		if len(m.filteredTasks) > contentH {
			pct := 0
			if denom := len(m.filteredTasks) - contentH; denom > 0 {
				pct = 100 * m.taskScroll / denom
			}
			lines = append(lines, mutedSt.Render(fmt.Sprintf(" scroll %d%%", pct)))
		}
	}

	lines = padLines(lines, iH, iW)

	box := borderStyle
	if m.activePane == paneTasks {
		box = activeBorderStyle
	}
	return box.Width(outerW).Height(outerH).Render(strings.Join(lines, "\n"))
}

func (m Model) renderTaskRow(task asana.Task, selected bool, w int) string {
	// Use plain ASCII to avoid Unicode width ambiguity in terminals
	status := "[ ]"
	if task.Completed {
		status = "[x]"
	}
	pri := "  "
	switch strings.ToLower(task.Priority) {
	case "high":
		pri = "Hi"
	case "medium":
		pri = "Md"
	case "low":
		pri = "Lo"
	}

	// 3 (status) + 1 (space) + 2 (pri) + 1 (space) + 1 (leading space) = 8
	maxName := w - 8
	if maxName < 1 {
		maxName = 1
	}
	name := task.Name
	runes := []rune(name)
	if len(runes) > maxName {
		name = string(runes[:maxName-1]) + "~"
	}

	line := fmt.Sprintf(" %s %s %s", status, pri, name)

	switch {
	case selected:
		return lipgloss.NewStyle().
			Background(colorSelected).Foreground(colorText).Bold(true).Width(w).Render(line)
	case task.Completed:
		return lipgloss.NewStyle().
			Foreground(colorMuted).Background(colorBg).Strikethrough(true).Width(w).Render(line)
	default:
		return lipgloss.NewStyle().
			Foreground(colorText).Background(colorBg).Width(w).Render(line)
	}
}

// ─── Task Detail Panel ───────────────────────────────────────────────────────

func (m Model) viewDetail(outerW, outerH int) string {
	iW := panelInnerW(outerW)
	iH := panelInnerH(outerH)

	titleSt := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).Bold(true).Width(iW)
	mutedSt := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).Width(iW)

	var lines []string
	lines = append(lines, titleSt.Render("Task Details"))

	if len(m.filteredTasks) == 0 || m.taskCursor >= len(m.filteredTasks) {
		lines = append(lines, mutedSt.Render(" Select a task"))
	} else {
		lines = append(lines, m.renderDetailLines(m.filteredTasks[m.taskCursor], iW)...)
	}

	lines = padLines(lines, iH, iW)
	return borderStyle.Width(outerW).Height(outerH).Render(strings.Join(lines, "\n"))
}

func (m Model) renderDetailLines(task asana.Task, w int) []string {
	var lines []string

	nameSt  := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBg).Bold(true).Width(w)
	keySt   := lipgloss.NewStyle().Foreground(colorYellow).Background(colorBg).Bold(true)
	valSt   := lipgloss.NewStyle().Foreground(colorText).Background(colorBg)
	blankSt := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).Width(w)

	lines = append(lines, nameSt.Render(task.Name))
	lines = append(lines, blankSt.Render(""))

	kv := func(k, v string) {
		if v == "" {
			return
		}
		lines = append(lines, keySt.Render(" "+k+": ")+valSt.Render(v))
	}

	kv("ID", task.GetID())
	if task.Completed {
		kv("Status", "Complete")
	} else {
		kv("Status", "Incomplete")
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
		lines = append(lines, blankSt.Render(""))
		lines = append(lines, keySt.Render(" Description:"))
		words := strings.Fields(task.Notes)
		cur := ""
		for _, wd := range words {
			if len(cur)+len(wd)+1 > w-3 {
				lines = append(lines, valSt.Render("  "+cur))
				cur = wd
			} else {
				if cur != "" {
					cur += " "
				}
				cur += wd
			}
		}
		if cur != "" {
			lines = append(lines, valSt.Render("  "+cur))
		}
	}
	return lines
}

// ─── Model / Provider Panel ──────────────────────────────────────────────────

func (m Model) viewModelPane(outerW, outerH int) string {
	iW := panelInnerW(outerW)
	iH := panelInnerH(outerH)

	active  := m.activePane == paneModel
	titleSt := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).Bold(true).Width(iW)

	sub := "Providers  [Tab->models]"
	if m.modelSubPane == 1 {
		sub = "Models  [Tab->providers]"
	}

	var lines []string
	lines = append(lines, titleSt.Render("Model > "+sub))

	if m.modelSubPane == 0 {
		for i, prov := range m.modelPane.providers {
			marker := "  "
			if prov.ID == m.modelPane.activeProvider {
				marker = "> "
			}
			label := marker + prov.Name
			switch {
			case i == m.modelPane.providerCursor && active:
				lines = append(lines, lipgloss.NewStyle().
					Background(colorSelected).Foreground(colorText).Bold(true).Width(iW).Render(label))
			case prov.ID == m.modelPane.activeProvider:
				lines = append(lines, lipgloss.NewStyle().
					Foreground(colorGreen).Background(colorBg).Bold(true).Width(iW).Render(label))
			default:
				lines = append(lines, lipgloss.NewStyle().
					Foreground(colorText).Background(colorBg).Width(iW).Render(label))
			}
		}
	} else {
		prov := m.modelPane.providers[m.modelPane.providerCursor]
		lines = append(lines, lipgloss.NewStyle().
			Foreground(colorMuted).Background(colorBg).Width(iW).Render(" "+prov.Name))
		for i, mod := range prov.Models {
			marker := "  "
			if mod == m.modelPane.activeModel && prov.ID == m.modelPane.activeProvider {
				marker = "> "
			}
			label := marker + mod
			switch {
			case i == m.modelPane.modelCursor && active:
				lines = append(lines, lipgloss.NewStyle().
					Background(colorSelected).Foreground(colorText).Bold(true).Width(iW).Render(label))
			case mod == m.modelPane.activeModel && prov.ID == m.modelPane.activeProvider:
				lines = append(lines, lipgloss.NewStyle().
					Foreground(colorGreen).Background(colorBg).Bold(true).Width(iW).Render(label))
			default:
				lines = append(lines, lipgloss.NewStyle().
					Foreground(colorText).Background(colorBg).Width(iW).Render(label))
			}
		}
	}

	lines = padLines(lines, iH, iW)

	box := borderStyle
	if active {
		box = activeBorderStyle
	}
	return box.Width(outerW).Height(outerH).Render(strings.Join(lines, "\n"))
}

// ─── Log Panel ───────────────────────────────────────────────────────────────

func (m Model) viewLogPanel(outerW, outerH int) string {
	iW := panelInnerW(outerW)
	iH := panelInnerH(outerH)

	titleSt := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).Bold(true).Width(iW)

	var lines []string
	lines = append(lines, titleSt.Render("Execution Log"))

	contentH := iH - 1
	if m.executing {
		contentH--
	}
	if contentH < 0 {
		contentH = 0
	}

	start := 0
	if len(m.logLines) > contentH {
		start = len(m.logLines) - contentH
	}

	for _, ll := range m.logLines[start:] {
		var fg lipgloss.Color
		switch ll.kind {
		case "ok":
			fg = colorGreen
		case "err":
			fg = colorRed
		case "dim":
			fg = colorMuted
		default:
			fg = colorText
		}
		lines = append(lines, lipgloss.NewStyle().
			Foreground(fg).Background(colorBg).Width(iW).Render(ll.text))
	}

	if m.executing {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(colorYellow).Background(colorBg).Width(iW).
			Render(m.spinner.View()+" Working..."))
	}

	lines = padLines(lines, iH, iW)
	return activeBorderStyle.Width(outerW).Height(outerH).Render(strings.Join(lines, "\n"))
}

// ─── Config Screen ───────────────────────────────────────────────────────────

func (m Model) viewConfigScreen() string {
	cardW := min(m.width-4, 72)
	maxFieldsVisible := (m.height - 4 - 3 - 2) / 5
	if maxFieldsVisible < 1 {
		maxFieldsVisible = 1
	}
	cardH := 3 + (maxFieldsVisible*5) + 3
	if cardH > m.height-2 {
		cardH = m.height - 2
	}

	iW     := panelInnerW(cardW)
	inputW := iW - 4

	var rows []string
	rows = append(rows,
		lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Background(colorBg).Render("Configuration"))
	rows = append(rows,
		lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).
			Render("Tab navigate  </> options  Ctrl-S save  Esc cancel"))
	rows = append(rows, "")

	fieldStart := 0
	if m.cfgCursor >= maxFieldsVisible {
		fieldStart = m.cfgCursor - maxFieldsVisible + 1
	}
	fieldEnd := min(fieldStart+maxFieldsVisible, len(configFields))

	for i := fieldStart; i < fieldEnd; i++ {
		f         := configFields[i]
		isFocused := i == m.cfgCursor

		var label string
		if isFocused {
			label = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Background(colorBg).
				Render("> " + f.label)
		} else {
			label = lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).
				Render("  " + f.label)
		}

		var valueStr string
		if len(f.options) > 0 {
			cursor := m.cfgOptCursors[i]
			var opts []string
			for j, opt := range f.options {
				if j == cursor {
					opts = append(opts, lipgloss.NewStyle().
						Foreground(colorYellow).Bold(true).Background(colorBg).Render("["+opt+"]"))
				} else {
					opts = append(opts, lipgloss.NewStyle().
						Foreground(colorMuted).Background(colorBg).Render(" "+opt+" "))
				}
			}
			valueStr = "  " + strings.Join(opts, " ")
		} else {
			inp := m.cfgInputs[i]
			if isFocused {
				valueStr = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent).
					Background(colorBg).Width(inputW).Render(inp.View())
			} else {
				display := inp.Value()
				if f.secret && display != "" {
					display = strings.Repeat("*", min(len(display), 32))
				}
				if display == "" {
					display = "(not set)"
				}
				valueStr = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder).
					Foreground(colorMuted).Background(colorBg).Width(inputW).Render(display)
			}
		}

		rows = append(rows, label)
		rows = append(rows, valueStr)
		rows = append(rows, "")
	}

	if len(configFields) > maxFieldsVisible {
		rows = append(rows, lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).
			Render(fmt.Sprintf("  field %d/%d", m.cfgCursor+1, len(configFields))))
	}

	for i, f := range configFields {
		if f.key == "provider" {
			chosen := f.options[m.cfgOptCursors[i]]
			if prov, ok := ai.GetProvider(chosen); ok {
				rows = append(rows, lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).
					Render("  Models: "+strings.Join(prov.Models, ", ")))
			}
			break
		}
	}

	card := activeBorderStyle.Width(cardW).Height(cardH).Render(strings.Join(rows, "\n"))
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center, card,
		lipgloss.WithWhitespaceBackground(colorBg))
}

// ─── Search Overlay ──────────────────────────────────────────────────────────

func (m Model) viewSearchOverlay() string {
	boxW := min(m.width-4, 64)
	content := lipgloss.NewStyle().Foreground(colorYellow).Background(colorBg).Bold(true).Render("/ ") +
		m.searchInput.View() +
		lipgloss.NewStyle().Foreground(colorMuted).Background(colorBg).Render("   Enter=search  Esc=cancel")
	box := lipgloss.NewStyle().
		Width(boxW).
		Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent).
		Background(colorBg).Padding(0, 1).
		Render(content)
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(colorBg))
}

// ─── Status Bar ──────────────────────────────────────────────────────────────

func (m Model) viewStatusBar() string {
	var fg lipgloss.Color
	switch m.statusKind {
	case "ok":
		fg = colorGreen
	case "err":
		fg = colorRed
	case "loading":
		fg = colorYellow
	default:
		fg = colorMuted
	}
	msg := m.statusMsg
	runes := []rune(msg)
	if len(runes) > m.width-2 {
		msg = string(runes[:m.width-5]) + "..."
	}
	return lipgloss.NewStyle().
		Width(m.width).Background(colorSurface).Foreground(fg).Bold(true).Padding(0, 1).
		Render(msg)
}

// ─── Keybind Bar ─────────────────────────────────────────────────────────────

func (m Model) viewKeybinds() string {
	binds := []struct{ k, d string }{
		{"jk", "nav"}, {"Enter", "execute"}, {"Tab", "pane"},
		{"/", "search"}, {"C", "config"}, {"L", "log"}, {"r", "refresh"}, {"q", "quit"},
	}
	if m.activePane == paneConfig {
		binds = []struct{ k, d string }{
			{"Tab", "navigate"}, {"<>", "option"}, {"Ctrl-S", "save"}, {"Esc", "cancel"},
		}
	}
	kSt  := lipgloss.NewStyle().Foreground(colorAccent).Background(colorSurface).Bold(true)
	dSt  := lipgloss.NewStyle().Foreground(colorMuted).Background(colorSurface)
	sep  := dSt.Render(" · ")
	var parts []string
	for _, b := range binds {
		parts = append(parts, kSt.Render(b.k)+dSt.Render(" "+b.d))
	}
	return lipgloss.NewStyle().Width(m.width).Background(colorSurface).
		Render(" " + strings.Join(parts, sep))
}

// ─── Run ─────────────────────────────────────────────────────────────────────

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