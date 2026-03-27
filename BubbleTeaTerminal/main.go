package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenInput screen = iota
	screenMode
	screenWizard
	screenForge
	screenStaging
)

type forgeDoneMsg struct{}
type forgeErrMsg struct{ message string }
type pollStatusMsg struct{}
type injectDoneMsg struct{ err error }
type bridgeStatusMsg struct{ alive bool }

type animTickMsg time.Time

type optionItem struct {
	title string
	desc  string
}

func (i optionItem) Title() string       { return i.title }
func (i optionItem) Description() string { return i.desc }
func (i optionItem) FilterValue() string { return i.title }

type craftedItem struct {
	label          string
	tier           string
	damageClass    string
	styleChoice    string
	projectile     string
	craftingStation string
}

type wizardStep struct {
	question string
	options  []optionItem
}

var wizardSteps = []wizardStep{
	{
		question: "Choose Tier",
		options: []optionItem{
			{title: "Starter", desc: "Early game balance and simple effects"},
			{title: "Dungeon", desc: "Midgame utility with stronger scaling"},
			{title: "Hardmode", desc: "High pressure combat and synergy hooks"},
			{title: "Endgame", desc: "Peak-tier stats and complex behavior"},
		},
	},
	{
		question: "Choose Class",
		options: []optionItem{
			{title: "Melee", desc: "Close-range burst and direct engagement"},
			{title: "Ranged", desc: "Projectile pressure from safe distance"},
			{title: "Magic", desc: "Mana-driven effects and spell identity"},
		},
	},
	{
		question: "Choose Style",
		options: []optionItem{
			{title: "Swing", desc: "Wide arc attacks for crowd control"},
			{title: "Stab", desc: "Precise thrust pattern with reach focus"},
			{title: "Hold", desc: "Channel behavior while key is held"},
		},
	},
	{
		question: "Choose Projectile",
		options: []optionItem{
			{title: "None", desc: "Purely melee interaction"},
			{title: "Standard Shot", desc: "Basic projectile companion attack"},
			{title: "Beam Slash", desc: "Arc beam emission on swing timing"},
			{title: "Thrown", desc: "Throwable behavior with return logic"},
		},
	},
	{
		question: "Choose Crafting Station",
		options: []optionItem{
			{title: "Auto", desc: "AI picks station based on tier and theme"},
			{title: "By Hand", desc: "No station required"},
			{title: "Workbench", desc: "Basic wood and early materials"},
			{title: "Iron Anvil", desc: "Pre-hardmode metal bars"},
			{title: "Mythril Anvil", desc: "Hardmode bars and components"},
			{title: "Ancient Manipulator", desc: "Lunar endgame fragments"},
		},
	},
}

type model struct {
	state  screen
	width  int
	height int

	craftedItems []craftedItem

	textInput  textinput.Model
	modeList   list.Model
	wizardList list.Model
	spinner    spinner.Model

	prompt          string
	tier            string
	damageClass     string
	styleChoice     string
	projectile      string
	craftingStation string

	wizardIndex   int
	errMsg        string
	injecting     bool
	animTick      int
	heat          int
	revealPhase   int
	lastForgeVerb int
	termCompact   bool

	forgeItemName  string // item name returned from the backend
	forgeErr       string // error message from the backend
	stageLabel     string // current pipeline stage label
	stageTargetPct int    // target heat % from pipeline status

	bridgeAlive bool   // forge_connector_alive.json present with live PID
	injectErr   string // non-empty if command_trigger write failed
}

const (
	compactWidthThreshold  = 84
	compactHeightThreshold = 24
)

var forgeVerbs = []string{"Tempering", "Binding", "Etching", "Awakening"}
var wizardGlyphs = []string{"\u26e8", "\u2694", "\u2736", "\u27b6"}

// ---------------------------------------------------------------------------
// Filesystem helpers – handshake with the Python orchestrator
// ---------------------------------------------------------------------------

func modSourcesDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "Terraria", "tModLoader", "ModSources")
}

func tierToKey(tier string) string {
	switch tier {
	case "Starter":
		return "Tier1_Starter"
	case "Dungeon":
		return "Tier2_Dungeon"
	case "Hardmode":
		return "Tier3_Hardmode"
	case "Endgame":
		return "Tier4_Endgame"
	default:
		return "Tier1_Starter" // "Auto" and anything unknown defaults to starter
	}
}

func writeUserRequest(prompt, tier, craftingStation string) error {
	dir := modSourcesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	payload := map[string]string{
		"prompt": prompt,
		"tier":   tierToKey(tier),
	}
	if craftingStation != "" && craftingStation != "Auto" {
		payload["crafting_station"] = craftingStation
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	dst := filepath.Join(dir, "user_request.json")
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

type pipelineStatus struct {
	status     string
	itemName   string
	errMsg     string
	stagePct   int
	stageLabel string
}

func readGenerationStatus() pipelineStatus {
	data, err := os.ReadFile(filepath.Join(modSourcesDir(), "generation_status.json"))
	if err != nil {
		return pipelineStatus{}
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return pipelineStatus{}
	}
	ps := pipelineStatus{}
	ps.status, _ = result["status"].(string)
	if batchList, ok := result["batch_list"].([]interface{}); ok && len(batchList) > 0 {
		ps.itemName, _ = batchList[0].(string)
	}
	ps.errMsg, _ = result["message"].(string)
	ps.stageLabel, _ = result["stage_label"].(string)
	if pct, ok := result["stage_pct"].(float64); ok {
		ps.stagePct = int(pct)
	}
	return ps
}

func pollStatusCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return pollStatusMsg{}
	})
}

// writeCommandTrigger atomically writes command_trigger.json to ModSources.
func writeCommandTrigger() error {
	dir := modSourcesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(map[string]string{
		"action":    "execute",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	dst := filepath.Join(dir, "command_trigger.json")
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// readHeartbeatFile returns true if the JSON heartbeat file exists,
// has status "listening", and its PID is still alive.
func readHeartbeatFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var hb map[string]interface{}
	if err := json.Unmarshal(data, &hb); err != nil {
		return false
	}
	if status, _ := hb["status"].(string); status != "listening" {
		return false
	}
	pidFloat, ok := hb["pid"].(float64)
	if !ok {
		return false
	}
	pid := int(pidFloat)
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal(0) checks liveness without sending a real signal.
	return proc.Signal(syscall.Signal(0)) == nil
}

// readBridgeHeartbeat returns true if forge_connector_alive.json exists
// and points at a live bridge process.
func readBridgeHeartbeat() bool {
	return readHeartbeatFile(filepath.Join(modSourcesDir(), "forge_connector_alive.json"))
}

// readOrchestratorHeartbeat returns true if orchestrator_alive.json exists
// and points at a live orchestrator process.
func readOrchestratorHeartbeat() bool {
	return readHeartbeatFile(filepath.Join(modSourcesDir(), "orchestrator_alive.json"))
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Ex: A void blade that eats light..."
	ti.Focus()
	ti.CharLimit = 120
	ti.Width = 54
	ti.Prompt = ""

	s := spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(colorRune)))

	delegate := list.NewDefaultDelegate()
	modeItems := []list.Item{
		optionItem{title: "Auto-Forge", desc: "AI decides balance & mechanics"},
		optionItem{title: "Manual Override", desc: "Configure tier, class, and style"},
	}
	modeList := list.New(modeItems, delegate, 56, 8)
	modeList.SetFilteringEnabled(false)
	modeList.SetShowHelp(false)
	modeList.SetShowStatusBar(false)
	modeList.SetShowPagination(false)
	modeList.DisableQuitKeybindings()
	modeList.Title = "Complexity Check"

	wizardList := list.New([]list.Item{}, delegate, 56, 8)
	wizardList.SetFilteringEnabled(false)
	wizardList.SetShowHelp(false)
	wizardList.SetShowStatusBar(false)
	wizardList.SetShowPagination(false)
	wizardList.DisableQuitKeybindings()
	wizardList.SetHeight(12)

	return model{
		state:      screenInput,
		textInput:  ti,
		modeList:   modeList,
		wizardList: wizardList,
		spinner:    s,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		animTickCmd(),
		tea.SetWindowTitle("The Forge"),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.termCompact = msg.Width < compactWidthThreshold || msg.Height < compactHeightThreshold
		panelWidth := max(56, msg.Width-10)
		if panelWidth > 88 {
			panelWidth = 88
		}
		m.modeList.SetWidth(panelWidth - 8)
		m.wizardList.SetWidth(panelWidth - 8)
	case animTickMsg:
		m.animTick++
		return m, animTickCmd()
	}

	switch m.state {
	case screenInput:
		return m.updateInput(msg)
	case screenMode:
		return m.updateMode(msg)
	case screenWizard:
		return m.updateWizard(msg)
	case screenForge:
		return m.updateForge(msg)
	case screenStaging:
		return m.updateStaging(msg)
	default:
		return m, nil
	}
}

func (m model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
		prompt := strings.TrimSpace(m.textInput.Value())
		if prompt == "" {
			m.errMsg = "Prompt cannot be empty."
			return m, nil
		}
		m.prompt = prompt
		m.errMsg = ""
		m.state = screenMode
		m.modeList.Select(0)
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) updateMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEsc:
			m.state = screenInput
			m.textInput.Focus()
			return m, nil
		case tea.KeyEnter:
			selected, _ := m.modeList.SelectedItem().(optionItem)
			if selected.title == "Manual Override" {
				m.wizardIndex = 0
				m.tier = ""
				m.damageClass = ""
				m.styleChoice = ""
				m.projectile = ""
				m.configureWizardStep()
				m.state = screenWizard
				return m, nil
			}
			m.tier = "Auto"
			m.damageClass = ""
			m.styleChoice = ""
			m.projectile = ""
			return m.enterForge()
		}
	}

	var cmd tea.Cmd
	m.modeList, cmd = m.modeList.Update(msg)
	return m, cmd
}

func (m model) updateWizard(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEsc:
			if m.wizardIndex == 0 {
				m.state = screenMode
				return m, nil
			}
			m.wizardIndex--
			switch m.wizardIndex {
			case 0:
				m.tier = ""
			case 1:
				m.damageClass = ""
			case 2:
				m.styleChoice = ""
			case 3:
				m.projectile = ""
			case 4:
				m.craftingStation = ""
			}
			m.configureWizardStep()
			return m, nil
		case tea.KeyEnter:
			selected, _ := m.wizardList.SelectedItem().(optionItem)
			switch m.wizardIndex {
			case 0:
				m.tier = selected.title
			case 1:
				m.damageClass = selected.title
			case 2:
				m.styleChoice = selected.title
			case 3:
				m.projectile = selected.title
			case 4:
				m.craftingStation = selected.title
			}
			m.wizardIndex++
			if m.wizardIndex >= len(wizardSteps) {
				return m.enterForge()
			}
			m.configureWizardStep()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.wizardList, cmd = m.wizardList.Update(msg)
	return m, cmd
}

func (m model) updateForge(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Allow escaping an error state.
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEsc && m.forgeErr != "" {
		m.state = screenInput
		m.forgeErr = ""
		m.textInput.Focus()
		return m, nil
	}

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.lastForgeVerb = m.animTick % len(forgeVerbs)
		// Animate heat smoothly toward the stage target.
		if m.heat < m.stageTargetPct {
			m.heat += 2
			if m.heat > m.stageTargetPct {
				m.heat = m.stageTargetPct
			}
		}
		return m, cmd
	case pollStatusMsg:
		ps := readGenerationStatus()
		switch ps.status {
		case "ready":
			m.forgeItemName = ps.itemName
			m.heat = 100
			return m, func() tea.Msg { return forgeDoneMsg{} }
		case "error":
			return m, func() tea.Msg { return forgeErrMsg{message: ps.errMsg} }
		default:
			// "building" or file not yet written — update stage and keep polling.
			if ps.stagePct > m.stageTargetPct {
				m.stageTargetPct = ps.stagePct
			}
			if ps.stageLabel != "" {
				m.stageLabel = ps.stageLabel
			}
			return m, pollStatusCmd()
		}
	case forgeErrMsg:
		m.forgeErr = msg.message
		return m, nil
	case forgeDoneMsg:
		m.state = screenStaging
		m.craftedItems = append(m.craftedItems, m.buildCraftedItem())
		m.injecting = false
		m.revealPhase = 1
		checkBridgeCmd := func() tea.Msg { return bridgeStatusMsg{alive: readBridgeHeartbeat()} }
		return m, tea.Batch(m.spinner.Tick, checkBridgeCmd)
	}
	return m, nil
}

func (m model) updateStaging(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.revealPhase < 3 {
			m.revealPhase++
			return m, m.spinner.Tick
		}
		return m, cmd
	case bridgeStatusMsg:
		m.bridgeAlive = msg.alive
		return m, nil
	case injectDoneMsg:
		m.injecting = false
		if msg.err != nil {
			m.injectErr = msg.err.Error()
		}
		return m, nil
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "c", "C":
			m.resetForCraftAnother()
			return m, nil
		case "enter":
			if m.injecting {
				return m, nil // debounce
			}
			m.injecting = true
			m.injectErr = ""
			injectCmd := func() tea.Msg {
				return injectDoneMsg{err: writeCommandTrigger()}
			}
			return m, injectCmd
		}
	}
	return m, nil
}

func (m model) View() string {
	content := m.screenView()
	panel := m.renderShell(content)

	if m.width <= 0 || m.height <= 0 {
		return panel
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		panel,
		lipgloss.WithWhitespaceBackground(colorBg),
	)
}

func (m model) screenView() string {
	switch m.state {
	case screenInput:
		return m.inputView()
	case screenMode:
		return m.modeView()
	case screenWizard:
		return m.wizardView()
	case screenForge:
		return m.forgeView()
	case screenStaging:
		return m.stagingView()
	default:
		return ""
	}
}

func (m model) inputView() string {
	lines := []string{
		styles.TitleRune.Render("The Forge :: Corruption Construct"),
		styles.Subtitle.Render("Describe the item you want to create"),
		"",
		styles.PromptInput.Render(m.textInput.View()),
	}
	if m.errMsg != "" {
		lines = append(lines, styles.Error.Render(m.errMsg))
	}
	lines = append(lines, "", styles.Hint.Render("Press Enter to continue"))
	return strings.Join(lines, "\n")
}

func (m model) modeView() string {
	return strings.Join([]string{
		m.modeList.View(),
		styles.Hint.Render("↑/↓ navigate  •  Enter select  •  Esc back"),
	}, "\n")
}

func (m model) wizardView() string {
	step := fmt.Sprintf("Step %d of %d", m.wizardIndex+1, len(wizardSteps))
	glyph := wizardGlyphs[m.wizardIndex%len(wizardGlyphs)]
	lines := []string{
		styles.TitleRune.Render(glyph + "  Manual Override"),
		styles.Progress.Render(step),
	}
	lines = append(lines, "", m.wizardList.View(), styles.Hint.Render("↑/↓ navigate  •  Enter select  •  Esc back"))
	return strings.Join(lines, "\n")
}

func (m model) forgeView() string {
	if m.forgeErr != "" {
		return strings.Join([]string{
			styles.Error.Render("✘ Forge Failed"),
			"",
			styles.Body.Render(m.forgeErr),
			"",
			styles.Hint.Render("Esc to go back"),
		}, "\n")
	}
	label := m.stageLabel
	if label == "" {
		label = forgeVerbs[m.lastForgeVerb%len(forgeVerbs)] + "..."
	}
	return strings.Join([]string{
		styles.TitleRune.Render("The Forge"),
		styles.Progress.Render("Heat " + m.heatBar()),
		"",
		fmt.Sprintf("%s %s", m.spinner.View(), styles.Subtitle.Render(label)),
		"",
		styles.Hint.Render("Calibrating corruption sigils and item logic"),
	}, "\n")
}

func (m model) stagingView() string {
	lines := []string{
		styles.Success.Render("✔ Item Ready"),
		styles.Subtitle.Render("Staging Area"),
		"",
	}

	if len(m.craftedItems) == 0 {
		lines = append(lines, styles.Hint.Render("No crafted items yet."))
	} else {
		for i, item := range m.craftedItems {
			lines = append(lines, styles.Inventory.Render(fmt.Sprintf("%d. %s", i+1, m.revealItem(item.label))))
			if m.revealPhase >= 3 && (item.damageClass != "" || item.styleChoice != "" || item.projectile != "") {
				meta := buildMetaLine(item)
				if meta != "" {
					lines = append(lines, styles.Meta.Render("   "+meta))
				}
			}
		}
	}

	lines = append(lines, "")

	// Bridge status line.
	if m.bridgeAlive {
		lines = append(lines, styles.Success.Render("⬡ Bridge Online"))
	} else {
		lines = append(lines, styles.Hint.Render("⬡ Bridge Offline — open Terraria with ForgeConnector loaded"))
	}

	if m.injectErr != "" {
		lines = append(lines, styles.Error.Render("✘ "+m.injectErr))
	}

	if m.injecting {
		lines = append(lines, "", styles.Injecting.Render("⟳ Injecting into Terraria..."))
	} else {
		lines = append(lines, "", styles.Hint.Render("[C] Craft Another   [ENTER] Execute"))
	}

	return strings.Join(lines, "\n")
}

func (m *model) configureWizardStep() {
	step := wizardSteps[m.wizardIndex]
	items := make([]list.Item, 0, len(step.options))
	for _, option := range step.options {
		items = append(items, option)
	}
	m.wizardList.SetItems(items)
	m.wizardList.Select(0)
	m.wizardList.SetHeight(max(12, len(step.options)*2+2))
	m.wizardList.Title = step.question
}

func (m model) enterForge() (tea.Model, tea.Cmd) {
	m.state = screenForge
	m.heat = 0
	m.stageTargetPct = 0
	m.stageLabel = ""
	m.animTick = 0
	m.lastForgeVerb = 0
	m.revealPhase = 0
	m.forgeErr = ""
	m.forgeItemName = ""

	prompt := m.prompt
	tier := m.tier
	craftingStation := m.craftingStation
	startCmd := func() tea.Msg {
		// Clear any stale status from a previous run.
		_ = os.Remove(filepath.Join(modSourcesDir(), "generation_status.json"))
		if err := writeUserRequest(prompt, tier, craftingStation); err != nil {
			return forgeErrMsg{message: "Failed to write request: " + err.Error()}
		}
		return pollStatusMsg{}
	}
	return m, tea.Batch(m.spinner.Tick, startCmd)
}

func animTickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

func (m *model) resetForCraftAnother() {
	m.state = screenInput
	m.prompt = ""
	m.tier = ""
	m.damageClass = ""
	m.styleChoice = ""
	m.projectile = ""
	m.wizardIndex = 0
	m.errMsg = ""
	m.injecting = false
	m.revealPhase = 0
	m.heat = 0
	m.forgeItemName = ""
	m.forgeErr = ""
	m.stageLabel = ""
	m.stageTargetPct = 0
	m.craftingStation = ""
	m.bridgeAlive = false
	m.injectErr = ""
	m.textInput.SetValue("")
	m.textInput.Focus()
	m.modeList.Select(0)
}

func (m model) buildCraftedItem() craftedItem {
	name := strings.TrimSpace(m.prompt)
	if name == "" {
		name = "Unnamed Artifact"
	}
	label := name
	if m.forgeItemName != "" {
		label = m.forgeItemName // real item name from backend
	} else if m.tier != "" {
		label = fmt.Sprintf("%s (%s)", name, m.tier)
	}
	return craftedItem{
		label:           label,
		tier:            m.tier,
		damageClass:     m.damageClass,
		styleChoice:     m.styleChoice,
		projectile:      m.projectile,
		craftingStation: m.craftingStation,
	}
}

func buildMetaLine(item craftedItem) string {
	parts := []string{}
	if item.damageClass != "" {
		parts = append(parts, item.damageClass)
	}
	if item.styleChoice != "" {
		parts = append(parts, item.styleChoice)
	}
	if item.projectile != "" && item.projectile != "None" {
		parts = append(parts, item.projectile)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " · ")
}

func (m model) renderShell(content string) string {
	ember := styles.Ember.Render(m.emberStrip())
	frame := styles.FrameCalm
	switch m.state {
	case screenWizard, screenMode:
		frame = styles.FrameCharged
	case screenForge, screenStaging:
		frame = styles.FrameCracked
	}

	panelBody := frame.Render(content)
	if m.termCompact {
		return strings.Join([]string{ember, panelBody}, "\n")
	}

	sigil := styles.SigilColumn.Render(m.sigilColumn())
	return strings.Join([]string{ember, lipgloss.JoinHorizontal(lipgloss.Top, panelBody, "  ", sigil)}, "\n")
}

func (m model) emberStrip() string {
	patterns := []string{
		"·  *   ·   +   ·  *",
		"*   ·   +   ·  *   ·",
		"+   ·  *   ·   +   ·",
	}
	return patterns[m.animTick%len(patterns)]
}

func (m model) sigilColumn() string {
	slots := []string{"Tier", "Class", "Style", "Proj", "Forge"}
	values := []string{m.tier, m.damageClass, m.styleChoice, m.projectile, m.craftingStation}
	lines := []string{styles.Meta.Render("Sigils")}
	for i := range slots {
		mark := "○"
		label := slots[i]
		if values[i] != "" {
			mark = "◉"
			label = values[i]
		}
		lines = append(lines, styles.Body.Render(fmt.Sprintf("%s %s", mark, label)))
	}
	return strings.Join(lines, "\n")
}

func (m model) heatBar() string {
	total := 12
	filled := (m.heat * total) / 100
	if filled > total {
		filled = total
	}
	empty := total - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty) + fmt.Sprintf(" %d%%", m.heat)
}

func (m model) revealItem(item string) string {
	switch {
	case m.revealPhase <= 0:
		return "..."
	case m.revealPhase == 1:
		return strings.Repeat("░", min(8, len(item)))
	case m.revealPhase == 2:
		n := len(item) / 2
		if n < 1 {
			n = 1
		}
		return item[:n] + "..."
	default:
		return item
	}
}

// findOrchestratorPath returns the path to orchestrator.py by checking
// locations relative to the binary and the working directory.
func findOrchestratorPath() string {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "..", "agents", "orchestrator.py"))
	}
	candidates = append(candidates,
		filepath.Join("..", "agents", "orchestrator.py"),
		"/Users/user/terraria/agents/orchestrator.py",
	)
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

func trimDotEnvComment(val string) string {
	inQuote := byte(0)
	escaped := false

	for i := 0; i < len(val); i++ {
		ch := val[i]

		if escaped {
			escaped = false
			continue
		}
		if inQuote != 0 {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}

		switch ch {
		case '"', '\'':
			inQuote = ch
		case '#':
			if i == 0 || val[i-1] == ' ' || val[i-1] == '\t' {
				return strings.TrimSpace(val[:i])
			}
		}
	}

	return strings.TrimSpace(val)
}

// parseDotEnv reads a .env file and returns key=value pairs.
// Handles quoted values and strips inline comments outside quotes.
func parseDotEnv(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var pairs []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eqIdx := strings.Index(line, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		val := strings.TrimSpace(line[eqIdx+1:])
		val = trimDotEnvComment(val)

		// Strip surrounding quotes.
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}

		pairs = append(pairs, key+"="+val)
	}
	return pairs
}

// ensureOrchestrator starts orchestrator.py if it is not already running.
// Logs are appended to orchestrator.log in the agents directory.
func ensureOrchestrator() {
	orchPath := findOrchestratorPath()
	if orchPath == "" {
		fmt.Fprintln(os.Stderr, "[forge] orchestrator.py not found — set FORGE_ORCHESTRATOR_PATH or run from the project root")
		return
	}

	// Skip only if the orchestrator heartbeat shows a live Python daemon.
	if readOrchestratorHeartbeat() {
		return
	}

	agentsDir := filepath.Dir(orchPath)
	logPath := filepath.Join(agentsDir, "orchestrator.log")
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	// Prefer the project venv's Python so all dependencies are available.
	python := filepath.Join(agentsDir, ".venv", "bin", "python3")
	if _, err := os.Stat(python); err != nil {
		python = "python3"
	}

	cmd := exec.Command(python, orchPath)
	cmd.Dir = agentsDir

	// Inherit current environment and inject .env vars so API keys are available.
	cmd.Env = append(os.Environ(), parseDotEnv(filepath.Join(agentsDir, ".env"))...)

	if logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[forge] failed to start orchestrator: %v\n", err)
	}
}

func main() {
	ensureOrchestrator()
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running forge ui: %v\n", err)
		os.Exit(1)
	}
}
