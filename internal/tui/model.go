// Package tui implements the Bubble Tea interactive interface.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thecoolrobot/task-agent/internal/ai"
	"github.com/thecoolrobot/task-agent/internal/asana"
	"github.com/thecoolrobot/task-agent/internal/config"
	"github.com/thecoolrobot/task-agent/internal/output"
)


// ─── Pane IDs ────────────────────────────────────────────────────────────────

type pane int

const (
	paneTasks pane = iota
	paneModel
	paneLog
	paneConfig // full-screen config editor
)

// ─── Messages ────────────────────────────────────────────────────────────────

type tasksLoadedMsg struct{ tasks []asana.Task }
type taskExecProgressMsg struct{ msg string }
type taskExecDoneMsg struct {
	result  *ai.TaskResult
	outPath string
	err     error
}
type searchDoneMsg struct{ tasks []asana.Task }
type errMsg struct{ err error }

// pollProgressCmd drains one message from the progress channel (non-blocking).
// Returns nil when the channel is empty so the program doesn't spin forever.
type progressTickMsg struct{ msg string }

func pollProgress(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil // channel closed, execution finished
		}
		return progressTickMsg{msg: msg}
	}
}

// ─── Config field descriptors ─────────────────────────────────────────────────

type configField struct {
	label    string
	key      string   // which cfg field
	secret   bool
	options  []string // if non-empty, render as a picker not a text input
	optionKey string  // e.g. "provider" or "model"
}

var configFields = []configField{
	{label: "Workspace GID",    key: "workspace_gid"},
	{label: "Project GID",      key: "project_gid"},
	{label: "Output directory", key: "output_dir"},
	{label: "Anthropic API key", key: "api_anthropic", secret: true},
	{label: "OpenAI API key",    key: "api_openai",    secret: true},
	{label: "Groq API key",      key: "api_groq",      secret: true},
	{label: "Moonshot API key",  key: "api_moonshot",  secret: true},
	{label: "AI Provider",       key: "provider",      options: []string{"anthropic", "openai", "groq", "ollama"}},
	{label: "Model",             key: "model"},
}

// ─── Model ───────────────────────────────────────────────────────────────────

// Model is the top-level Bubble Tea model.
type Model struct {
	width, height int

	cfg         *config.Config
	asanaClient *asana.Client

	// Task list
	tasks         []asana.Task
	filteredTasks []asana.Task
	taskCursor    int
	taskScroll    int

	// Pane navigation
	activePane pane

	// Model/provider switcher panel
	modelPane    modelPaneState
	modelSubPane int // 0=providers 1=models

	// Execution log
	logLines    []logLine
	progressCh  <-chan string // live AI progress feed

	// Search
	searchInput textinput.Model
	searching   bool

	// Config screen
	cfgCursor   int            // which field is focused
	cfgInputs   []textinput.Model
	cfgOptCursors []int        // per-option-field cursor

	// Spinner
	spinner   spinner.Model
	loading   bool
	executing bool

	// Status bar
	statusMsg  string
	statusKind string // "ok" | "err" | "loading"
}

type modelPaneState struct {
	providers      []ai.Provider
	providerCursor int
	modelCursor    int
	activeProvider string
	activeModel    string
}

type logLine struct {
	text string
	kind string // "info" | "ok" | "err" | "dim"
}

// ─── Constructor ─────────────────────────────────────────────────────────────

func New(cfg *config.Config, client *asana.Client) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorAccent)

	si := textinput.New()
	si.Placeholder = "search tasks..."
	si.CharLimit = 100

	// Build config inputs
	cfgInputs := make([]textinput.Model, len(configFields))
	cfgOptCursors := make([]int, len(configFields))
	for i, f := range configFields {
		ti := textinput.New()
		ti.CharLimit = 256
		ti.Placeholder = f.label
		if f.secret {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}
		// Populate initial values
		switch f.key {
		case "workspace_gid":
			ti.SetValue(cfg.WorkspaceGID)
		case "project_gid":
			ti.SetValue(cfg.ProjectGID)
		case "output_dir":
			ti.SetValue(cfg.OutputDir)
		case "api_anthropic":
			ti.SetValue(cfg.APIKeys["anthropic"])
		case "api_openai":
			ti.SetValue(cfg.APIKeys["openai"])
		case "api_groq":
			ti.SetValue(cfg.APIKeys["groq"])
		case "api_moonshot":
			ti.SetValue(cfg.APIKeys["moonshot"])
		case "provider":
			// find cursor for current provider
			for j, opt := range f.options {
				if opt == cfg.Provider {
					cfgOptCursors[i] = j
				}
			}
		case "model":
			// will be populated dynamically
			ti.SetValue(cfg.Model)
		}
		cfgInputs[i] = ti
	}

	// Find initial model cursor for provider
	provCursor, modelCursor := 0, 0
	for i, p := range ai.Providers {
		if p.ID == cfg.Provider {
			provCursor = i
			for j, m := range p.Models {
				if m == cfg.Model {
					modelCursor = j
				}
			}
		}
	}

	return Model{
		cfg:           cfg,
		asanaClient:   client,
		activePane:    paneTasks,
		spinner:       sp,
		searchInput:   si,
		cfgInputs:     cfgInputs,
		cfgOptCursors: cfgOptCursors,
		statusMsg:     "Loading tasks...",
		statusKind:    "loading",
		loading:       true,
		modelPane: modelPaneState{
			providers:      ai.Providers,
			providerCursor: provCursor,
			modelCursor:    modelCursor,
			activeProvider: cfg.Provider,
			activeModel:    cfg.Model,
		},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.cmdLoadTasks())
}

// ─── Load tasks ───────────────────────────────────────────────────────────────

func (m Model) cmdLoadTasks() tea.Cmd {
	return func() tea.Msg {
		if m.asanaClient == nil {
			return tasksLoadedMsg{tasks: []asana.Task{}}
		}
		tasks, err := m.asanaClient.ListTasks(m.cfg.ProjectGID)
		if err != nil {
			return errMsg{err}
		}
		return tasksLoadedMsg{tasks: tasks}
	}
}

// ─── Update ──────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tasksLoadedMsg:
		m.tasks = msg.tasks
		m.filteredTasks = msg.tasks
		m.loading = false
		m.statusMsg = fmt.Sprintf("Loaded %d tasks  [↑↓ navigate · Enter execute · Tab switch pane · C config · ? help]", len(m.tasks))
		m.statusKind = "ok"

	case searchDoneMsg:
		m.filteredTasks = msg.tasks
		m.taskCursor, m.taskScroll = 0, 0
		m.searching = false
		m.activePane = paneTasks
		m.statusMsg = fmt.Sprintf("Found %d tasks", len(m.filteredTasks))
		m.statusKind = "ok"

	case progressTickMsg:
		// A live progress message arrived — append and poll again
		m.logLines = append(m.logLines, logLine{text: "  " + msg.msg, kind: "dim"})
		if m.progressCh != nil {
			cmds = append(cmds, pollProgress(m.progressCh))
		}

	case taskExecDoneMsg:
		m.executing = false
		m.loading = false
		m.progressCh = nil
		if msg.err != nil {
			m.logLines = append(m.logLines, logLine{text: "❌  " + msg.err.Error(), kind: "err"})
			m.statusMsg = "Execution failed — press Esc to return"
			m.statusKind = "err"
		} else {
			m.logLines = append(m.logLines, logLine{text: "✅  Done!", kind: "ok"})
			m.logLines = append(m.logLines, logLine{text: "📁  " + msg.outPath, kind: "ok"})
			m.logLines = append(m.logLines, logLine{text: "", kind: "info"})
			for _, line := range strings.Split(output.Preview(msg.result), "\n") {
				m.logLines = append(m.logLines, logLine{text: line, kind: "info"})
			}
			m.statusMsg = "✅ Task complete — output saved to " + msg.outPath
			m.statusKind = "ok"
		}

	case errMsg:
		m.loading = false
		m.executing = false
		m.progressCh = nil
		m.statusMsg = "Error: " + msg.err.Error()
		m.statusKind = "err"
		cmds = append(cmds, m.spinner.Tick)
	}

	// Keep spinner ticking while loading/executing
	if (m.loading || m.executing) && len(cmds) == 0 {
		cmds = append(cmds, m.spinner.Tick)
	}

	return m, tea.Batch(cmds...)
}

// ─── Key handling ─────────────────────────────────────────────────────────────

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {

	// ── Search mode ──────────────────────────────────────────────────────────
	if m.searching {
		switch msg.Type {
		case tea.KeyEscape:
			m.searching = false
			m.searchInput.Blur()
			m.activePane = paneTasks
		case tea.KeyEnter:
			q := strings.TrimSpace(m.searchInput.Value())
			m.searchInput.Blur()
			m.searching = false
			if q == "" {
				m.filteredTasks = m.tasks
				m.activePane = paneTasks
				return m, nil
			}
			return m, m.cmdSearch(q)
		default:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// ── Config screen ────────────────────────────────────────────────────────
	if m.activePane == paneConfig {
		return m.handleConfigKey(msg)
	}

	// ── Global keys ──────────────────────────────────────────────────────────
	switch msg.String() {
	case "q", "ctrl+c":
		_ = config.Save(m.cfg)
		return m, tea.Quit

	case "esc":
		m.activePane = paneTasks

	case "tab":
		m.cyclePane()

	case "up", "k":
		m.cursorUp()

	case "down", "j":
		m.cursorDown()

	case "enter":
		return m.handleEnter()

	case "/":
		m.searching = true
		m.searchInput.SetValue("")
		m.searchInput.Focus()

	case "r":
		m.loading = true
		m.statusMsg = "Refreshing..."
		m.statusKind = "loading"
		return m, tea.Batch(m.cmdLoadTasks(), m.spinner.Tick)

	case "c", "C":
		m.refreshConfigInputs()
		m.cfgCursor = 0
		m.cfgInputs[0].Focus()
		m.activePane = paneConfig

	case "l", "L":
		m.activePane = paneLog
	}

	return m, nil
}

func (m Model) handleConfigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	f := configFields[m.cfgCursor]

	switch msg.Type {
	case tea.KeyEscape:
		m.cfgInputs[m.cfgCursor].Blur()
		m.activePane = paneTasks
		return m, nil

	case tea.KeyTab, tea.KeyShiftTab:
		// Move between fields
		m.cfgInputs[m.cfgCursor].Blur()
		if msg.Type == tea.KeyTab {
			m.cfgCursor = (m.cfgCursor + 1) % len(configFields)
		} else {
			m.cfgCursor = (m.cfgCursor - 1 + len(configFields)) % len(configFields)
		}
		m.cfgInputs[m.cfgCursor].Focus()
		return m, nil

	case tea.KeyEnter:
		if len(f.options) > 0 {
			// Cycle option
			m.cfgOptCursors[m.cfgCursor] = (m.cfgOptCursors[m.cfgCursor] + 1) % len(f.options)
		} else {
			// Save text field and advance
			m.saveConfigField(m.cfgCursor)
			m.cfgInputs[m.cfgCursor].Blur()
			m.cfgCursor = (m.cfgCursor + 1) % len(configFields)
			m.cfgInputs[m.cfgCursor].Focus()
		}
		return m, nil

	case tea.KeyCtrlS:
		// Save all and exit config
		m.saveAllConfigFields()
		if err := config.Save(m.cfg); err != nil {
			m.statusMsg = "❌ Save failed: " + err.Error()
			m.statusKind = "err"
		} else {
			m.statusMsg = "✅ Config saved to ~/.task-agent/config.json"
			m.statusKind = "ok"
		}
		m.cfgInputs[m.cfgCursor].Blur()
		m.activePane = paneTasks
		// Update model pane to reflect new provider/model
		m.refreshModelPane()
		return m, nil

	case tea.KeyLeft:
		if len(f.options) > 0 {
			n := len(f.options)
			m.cfgOptCursors[m.cfgCursor] = (m.cfgOptCursors[m.cfgCursor] - 1 + n) % n
		}
		return m, nil

	case tea.KeyRight:
		if len(f.options) > 0 {
			m.cfgOptCursors[m.cfgCursor] = (m.cfgOptCursors[m.cfgCursor] + 1) % len(f.options)
		}
		return m, nil
	}

	// Normal text input
	if len(f.options) == 0 {
		var cmd tea.Cmd
		m.cfgInputs[m.cfgCursor], cmd = m.cfgInputs[m.cfgCursor].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) saveConfigField(idx int) {
	f := configFields[idx]
	val := strings.TrimSpace(m.cfgInputs[idx].Value())
	switch f.key {
	case "workspace_gid":
		m.cfg.WorkspaceGID = val
	case "project_gid":
		m.cfg.ProjectGID = val
	case "output_dir":
		m.cfg.OutputDir = val
	case "api_anthropic":
		config.SetAPIKey(m.cfg, "anthropic", val)
	case "api_openai":
		config.SetAPIKey(m.cfg, "openai", val)
	case "api_groq":
		config.SetAPIKey(m.cfg, "groq", val)
	case "api_moonshot":
		config.SetAPIKey(m.cfg, "moonshot", val)
	}
}

func (m *Model) saveAllConfigFields() {
	for i, f := range configFields {
		if len(f.options) > 0 {
			switch f.key {
			case "provider":
				m.cfg.Provider = f.options[m.cfgOptCursors[i]]
				// Reset model when provider changes
				if provDef, ok := ai.GetProvider(m.cfg.Provider); ok {
					m.cfg.Model = provDef.DefaultModel
				}
			}
		} else {
			m.saveConfigField(i)
		}
	}
}

func (m *Model) refreshConfigInputs() {
	vals := map[string]string{
		"workspace_gid": m.cfg.WorkspaceGID,
		"project_gid":   m.cfg.ProjectGID,
		"output_dir":    m.cfg.OutputDir,
		"api_anthropic": m.cfg.APIKeys["anthropic"],
		"api_openai":    m.cfg.APIKeys["openai"],
		"api_groq":      m.cfg.APIKeys["groq"],
		"api_moonshot":  m.cfg.APIKeys["moonshot"],
		"model":         m.cfg.Model,
	}
	for i, f := range configFields {
		if v, ok := vals[f.key]; ok {
			m.cfgInputs[i].SetValue(v)
		}
		if f.key == "provider" {
			for j, opt := range f.options {
				if opt == m.cfg.Provider {
					m.cfgOptCursors[i] = j
				}
			}
		}
	}
}

func (m *Model) refreshModelPane() {
	for i, p := range ai.Providers {
		if p.ID == m.cfg.Provider {
			m.modelPane.providerCursor = i
			m.modelPane.activeProvider = p.ID
			for j, mod := range p.Models {
				if mod == m.cfg.Model {
					m.modelPane.modelCursor = j
				}
			}
			m.modelPane.activeModel = m.cfg.Model
		}
	}
}

// ─── Navigation helpers ───────────────────────────────────────────────────────

func (m *Model) cyclePane() {
	switch m.activePane {
	case paneTasks:
		m.activePane = paneModel
		m.modelSubPane = 0
	case paneModel:
		if m.modelSubPane == 0 {
			m.modelSubPane = 1
		} else {
			m.modelSubPane = 0
			m.activePane = paneLog
		}
	case paneLog:
		m.activePane = paneTasks
	}
}

func (m *Model) cursorUp() {
	switch m.activePane {
	case paneTasks:
		if m.taskCursor > 0 {
			m.taskCursor--
			if m.taskCursor < m.taskScroll {
				m.taskScroll--
			}
		}
	case paneModel:
		if m.modelSubPane == 0 && m.modelPane.providerCursor > 0 {
			m.modelPane.providerCursor--
		} else if m.modelSubPane == 1 && m.modelPane.modelCursor > 0 {
			m.modelPane.modelCursor--
		}
	}
}

func (m *Model) cursorDown() {
	switch m.activePane {
	case paneTasks:
		if m.taskCursor < len(m.filteredTasks)-1 {
			m.taskCursor++
			visH := m.height - 6
			if m.taskCursor >= m.taskScroll+visH {
				m.taskScroll++
			}
		}
	case paneModel:
		if m.modelSubPane == 0 {
			if m.modelPane.providerCursor < len(m.modelPane.providers)-1 {
				m.modelPane.providerCursor++
			}
		} else {
			prov := m.modelPane.providers[m.modelPane.providerCursor]
			if m.modelPane.modelCursor < len(prov.Models)-1 {
				m.modelPane.modelCursor++
			}
		}
	}
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.activePane {
	case paneTasks:
		if len(m.filteredTasks) == 0 {
			return m, nil
		}
		return m.executeTask(m.filteredTasks[m.taskCursor])

	case paneModel:
		if m.modelSubPane == 0 {
			m.modelSubPane = 1
			m.modelPane.modelCursor = 0
		} else {
			prov := m.modelPane.providers[m.modelPane.providerCursor]
			mod := prov.Models[m.modelPane.modelCursor]
			m.modelPane.activeProvider = prov.ID
			m.modelPane.activeModel = mod
			m.cfg.Provider = prov.ID
			m.cfg.Model = mod
			m.statusMsg = fmt.Sprintf("✅ Switched to %s / %s", prov.Name, mod)
			m.statusKind = "ok"
			m.modelSubPane = 0
		}
	}
	return m, nil
}

// ─── Task execution with live streaming ───────────────────────────────────────

func (m Model) executeTask(task asana.Task) (tea.Model, tea.Cmd) {
	m.executing = true
	m.loading = true
	m.logLines = []logLine{
		{text: fmt.Sprintf("⚡  Executing: %s", task.Name), kind: "info"},
		{text: fmt.Sprintf("🤖  Provider : %s / %s", m.cfg.Provider, m.cfg.Model), kind: "dim"},
		{text: "", kind: "info"},
	}
	m.activePane = paneLog
	m.statusMsg = "Running in YOLO mode..."
	m.statusKind = "loading"

	// Buffered channel for streaming progress
	ch := make(chan string, 64)
	m.progressCh = ch

	apiKey := config.GetAPIKey(m.cfg, m.cfg.Provider)
	providerID := m.cfg.Provider
	model := m.cfg.Model
	outDir := m.cfg.OutputDir
	if outDir == "" {
		outDir = "./task-outputs"
	}

	execCmd := func() tea.Msg {
		client := ai.NewClient(providerID, model, apiKey)
		taskMD := asana.FormatTaskMarkdown(&task)
		result, err := client.ExecuteTask(taskMD, func(s string) {
			ch <- s
		})
		close(ch)
		if err != nil {
			return taskExecDoneMsg{err: err}
		}
		outPath, err := output.Write(result, &task, outDir)
		if err != nil {
			return taskExecDoneMsg{result: result, err: err}
		}
		return taskExecDoneMsg{result: result, outPath: outPath}
	}

	return m, tea.Batch(
		m.spinner.Tick,
		execCmd,
		pollProgress(ch),
	)
}

// ─── Search ───────────────────────────────────────────────────────────────────

func (m Model) cmdSearch(query string) tea.Cmd {
	return func() tea.Msg {
		wsGID := m.cfg.WorkspaceGID
		if wsGID != "" && m.asanaClient != nil {
			tasks, err := m.asanaClient.SearchTasks(wsGID, query)
			if err == nil {
				return searchDoneMsg{tasks: tasks}
			}
		}
		// local filter fallback
		q := strings.ToLower(query)
		var filtered []asana.Task
		for _, t := range m.tasks {
			if strings.Contains(strings.ToLower(t.Name), q) ||
				strings.Contains(strings.ToLower(t.Notes), q) {
				filtered = append(filtered, t)
			}
		}
		return searchDoneMsg{tasks: filtered}
	}
}