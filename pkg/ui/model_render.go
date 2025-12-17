package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderQuitConfirm() string {
	t := m.theme

	boxStyle := t.Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Blocked).
		Padding(1, 3).
		Align(lipgloss.Center)

	titleStyle := t.Renderer.NewStyle().
		Foreground(t.Blocked).
		Bold(true)

	textStyle := t.Renderer.NewStyle().
		Foreground(t.Base.GetForeground())

	keyStyle := t.Renderer.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	content := titleStyle.Render("Quit bv?") + "\n\n" +
		textStyle.Render("Press ") + keyStyle.Render("Esc") + textStyle.Render(" or ") + keyStyle.Render("Y") + textStyle.Render(" to quit\n") +
		textStyle.Render("Press any other key to cancel")

	box := boxStyle.Render(content)

	return lipgloss.Place(
		m.width,
		m.height-1,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

func (m Model) renderListWithHeader() string {
	t := m.theme

	// Calculate dimensions based on actual list height set in sizing
	availableHeight := m.list.Height()
	if availableHeight == 0 {
		availableHeight = m.height - 3 // fallback
	}

	// Render column header
	headerStyle := t.Renderer.NewStyle().
		Background(t.Primary).
		Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#282A36"}).
		Bold(true).
		Width(m.width - 2)

	headerText := "  TYPE PRI STATUS      ID                                   TITLE"
	if m.workspaceMode {
		// Account for repo badges like [API] shown in workspace mode.
		headerText = "  REPO TYPE PRI STATUS      ID                               TITLE"
	}
	header := headerStyle.Render(headerText)

	// Page info
	totalItems := len(m.list.Items())
	currentIdx := m.list.Index()
	itemsPerPage := availableHeight
	if itemsPerPage < 1 {
		itemsPerPage = 1
	}
	currentPage := (currentIdx / itemsPerPage) + 1
	totalPages := (totalItems + itemsPerPage - 1) / itemsPerPage
	if totalPages < 1 {
		totalPages = 1
	}
	startItem := 0
	endItem := 0
	if totalItems > 0 {
		startItem = (currentPage-1)*itemsPerPage + 1
		endItem = startItem + itemsPerPage - 1
		if endItem > totalItems {
			endItem = totalItems
		}
	}

	pageInfo := fmt.Sprintf(" Page %d of %d (items %d-%d of %d) ", currentPage, totalPages, startItem, endItem, totalItems)
	pageStyle := t.Renderer.NewStyle().
		Foreground(t.Secondary).
		Align(lipgloss.Right).
		Width(m.width - 2)

	// Combine header with page info on the right
	headerLine := lipgloss.JoinHorizontal(lipgloss.Top,
		header,
	)

	// List view - just render it normally since bubbles handles scrolling
	listView := m.list.View()

	// Page indicator line
	pageLine := pageStyle.Render(pageInfo)

	// Combine all elements and force exact height
	// bodyHeight = m.height - 1 (1 for footer)
	bodyHeight := m.height - 1
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	// Build content with explicit height constraint
	// Header (1) + List + PageLine (1) must fit in bodyHeight
	content := lipgloss.JoinVertical(lipgloss.Left, headerLine, listView, pageLine)

	// Force exact height to prevent overflow
	return lipgloss.NewStyle().
		Width(m.width).
		Height(bodyHeight).
		MaxHeight(bodyHeight).
		Render(content)
}

func (m Model) renderSplitView() string {
	t := m.theme

	var listStyle, detailStyle lipgloss.Style

	if m.focused == focusList {
		listStyle = FocusedPanelStyle
		detailStyle = PanelStyle
	} else {
		listStyle = PanelStyle
		detailStyle = FocusedPanelStyle
	}

	// m.list.Width() is the inner width (set in Update)
	listInnerWidth := m.list.Width()
	panelHeight := m.height - 1

	// Create header row for list
	headerStyle := t.Renderer.NewStyle().
		Background(t.Primary).
		Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#282A36"}).
		Bold(true).
		Width(listInnerWidth)

	header := headerStyle.Render("  TYPE PRI STATUS      ID                     TITLE")

	// Page info for list
	totalItems := len(m.list.Items())
	currentIdx := m.list.Index()
	listHeight := m.list.Height()
	if listHeight == 0 {
		listHeight = panelHeight - 3 // fallback
	}
	if listHeight < 1 {
		listHeight = 1
	}
	currentPage := (currentIdx / listHeight) + 1
	totalPages := (totalItems + listHeight - 1) / listHeight
	if totalPages < 1 {
		totalPages = 1
	}
	startItem := 0
	endItem := 0
	if totalItems > 0 {
		startItem = (currentPage-1)*listHeight + 1
		endItem = startItem + listHeight - 1
		if endItem > totalItems {
			endItem = totalItems
		}
	}

	pageInfo := fmt.Sprintf("Page %d/%d (%d-%d of %d) ", currentPage, totalPages, startItem, endItem, totalItems)
	pageStyle := t.Renderer.NewStyle().
		Foreground(t.Secondary).
		Width(listInnerWidth).
		Align(lipgloss.Center)

	pageLine := pageStyle.Render(pageInfo)

	// Combine header + list + page indicator
	listContent := lipgloss.JoinVertical(lipgloss.Left, header, m.list.View(), pageLine)

	// List Panel Width: Inner + 2 (Padding). Border adds another 2.
	// Use MaxHeight to ensure content doesn't overflow
	listView := listStyle.
		Width(listInnerWidth + 2).
		Height(panelHeight).
		MaxHeight(panelHeight).
		Render(listContent)

	// Detail Panel Width: Inner + 2 (Padding). Border adds another 2.
	detailView := detailStyle.
		Width(m.viewport.Width + 2).
		Height(panelHeight).
		MaxHeight(panelHeight).
		Render(m.viewport.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m *Model) renderHelpOverlay() string {
	t := m.theme

	titleStyle := t.Renderer.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		MarginBottom(1)

	sectionStyle := t.Renderer.NewStyle().
		Foreground(t.Secondary).
		Bold(true).
		MarginTop(1)

	keyStyle := t.Renderer.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#7D56F4", Dark: "#BD93F9"}).
		Bold(true).
		Width(12)

	descStyle := t.Renderer.NewStyle().
		Foreground(t.Base.GetForeground())

	var sb strings.Builder

	sb.WriteString(titleStyle.Render("⌨️  Keyboard Shortcuts"))
	sb.WriteString("\n\n")

	// Navigation
	sb.WriteString(sectionStyle.Render("Navigation"))
	sb.WriteString("\n")
	shortcuts := []struct{ key, desc string }{
		{"j / ↓", "Move down"},
		{"k / ↑", "Move up"},
		{"home", "Go to first item"},
		{"G / end", "Go to last item"},
		{"Ctrl+d", "Page down"},
		{"Ctrl+u", "Page up"},
		{"Tab", "Switch focus (split view)"},
		{"Enter", "View details"},
		{"Esc", "Back / close"},
	}
	for _, s := range shortcuts {
		sb.WriteString(keyStyle.Render(s.key) + descStyle.Render(s.desc) + "\n")
	}

	// Views
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("Views"))
	sb.WriteString("\n")
	views := []struct{ key, desc string }{
		{"a", "Toggle Actionable view"},
		{"b", "Toggle Kanban board"},
		{"g", "Toggle Graph view"},
		{"H", "Toggle History view"},
		{"i", "Toggle Insights dashboard"},
		{"P", "Toggle Sprint dashboard"},
		{"R", "Open Recipe picker"},
		{"w", "Repo filter (workspace mode)"},
		{"?", "Toggle this help"},
	}
	for _, s := range views {
		sb.WriteString(keyStyle.Render(s.key) + descStyle.Render(s.desc) + "\n")
	}

	// Graph view keys
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("Graph View"))
	sb.WriteString("\n")
	graphKeys := []struct{ key, desc string }{
		{"hjkl", "Navigate nodes"},
		{"H/L", "Scroll canvas left/right"},
		{"PgUp/PgDn", "Scroll canvas up/down"},
		{"Enter", "Jump to selected issue"},
	}
	for _, s := range graphKeys {
		sb.WriteString(keyStyle.Render(s.key) + descStyle.Render(s.desc) + "\n")
	}

	// Insights (when in insights view)
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("Insights Panel"))
	sb.WriteString("\n")
	insightsKeys := []struct{ key, desc string }{
		{"h/l/Tab", "Switch metric panels"},
		{"j/k", "Navigate items"},
		{"e", "Toggle explanations"},
		{"x", "Toggle calculation details"},
		{"Enter", "Jump to issue"},
	}
	for _, s := range insightsKeys {
		sb.WriteString(keyStyle.Render(s.key) + descStyle.Render(s.desc) + "\n")
	}

	// History View keys
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("History View"))
	sb.WriteString("\n")
	historyKeys := []struct{ key, desc string }{
		{"j/k", "Navigate bead list"},
		{"J/K", "Navigate commits in bead"},
		{"Tab", "Toggle list/detail focus"},
		{"Enter", "Jump to selected bead"},
		{"y", "Copy commit SHA"},
		{"c", "Cycle confidence filter"},
		{"H/Esc", "Close history view"},
	}
	for _, s := range historyKeys {
		sb.WriteString(keyStyle.Render(s.key) + descStyle.Render(s.desc) + "\n")
	}

	// Filters
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("Filters"))
	sb.WriteString("\n")
	filters := []struct{ key, desc string }{
		{"o", "Show Open issues"},
		{"c", "Show Closed issues"},
		{"r", "Show Ready (unblocked)"},
		{"a", "Show All issues"},
		{"/", "Fuzzy search"},
		{"Ctrl+S", "Toggle semantic search mode"},
	}
	for _, s := range filters {
		sb.WriteString(keyStyle.Render(s.key) + descStyle.Render(s.desc) + "\n")
	}

	// General
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("General"))
	sb.WriteString("\n")
	general := []struct{ key, desc string }{
		{"t", "Time-travel (custom revision)"},
		{"T", "Time-travel (HEAD~5)"},
		{"E", "Export to Markdown"},
		{"C", "Copy issue to clipboard"},
		{"O", "Open in editor"},
		{"q", "Back / Quit"},
		{"Ctrl+c", "Force quit"},
	}
	for _, s := range general {
		sb.WriteString(keyStyle.Render(s.key) + descStyle.Render(s.desc) + "\n")
	}

	// Build full content (without footer yet)
	fullContent := sb.String()
	lines := strings.Split(fullContent, "\n")
	totalLines := len(lines)

	// Calculate available height for content
	// Reserve space for border (2), padding (2), and footer (2)
	availableHeight := m.height - 8
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Clamp scroll position
	maxScroll := totalLines - availableHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.helpScroll > maxScroll {
		m.helpScroll = maxScroll
	}
	if m.helpScroll < 0 {
		m.helpScroll = 0
	}

	// Extract visible lines
	startLine := m.helpScroll
	endLine := startLine + availableHeight
	if endLine > totalLines {
		endLine = totalLines
	}
	visibleLines := lines[startLine:endLine]
	visibleContent := strings.Join(visibleLines, "\n")

	// Build scroll indicator
	var scrollIndicator string
	if totalLines > availableHeight {
		// Show scroll position as a bar
		scrollPercent := 0
		if maxScroll > 0 {
			scrollPercent = m.helpScroll * 100 / maxScroll
		}
		// Create a simple indicator: [===---] style
		barWidth := 10
		filledWidth := barWidth * scrollPercent / 100
		if filledWidth > barWidth {
			filledWidth = barWidth
		}
		scrollBar := strings.Repeat("─", filledWidth) + "●" + strings.Repeat("─", barWidth-filledWidth)
		scrollIndicator = fmt.Sprintf(" [%s] %d/%d", scrollBar, m.helpScroll+1, maxScroll+1)
	}

	// Build footer with navigation hint
	footerStyle := t.Renderer.NewStyle().Foreground(t.Secondary).Italic(true)
	var footer string
	if totalLines > availableHeight {
		footer = footerStyle.Render(fmt.Sprintf("↑↓ scroll │ q close%s", scrollIndicator))
	} else {
		footer = footerStyle.Render("Press any key to close")
	}

	// Combine content and footer
	helpContent := visibleContent + "\n\n" + footer

	// Center the help content
	helpBox := t.Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 3).
		Render(helpContent)

	// Center in viewport
	return lipgloss.Place(
		m.width,
		m.height-1,
		lipgloss.Center,
		lipgloss.Center,
		helpBox,
	)
}

func (m *Model) renderFooter() string {
	// ══════════════════════════════════════════════════════════════════════════
	// POLISHED FOOTER - Stripe-level status bar with visual hierarchy
	// ══════════════════════════════════════════════════════════════════════════

	// If there's a status message, show it prominently with polished styling
	if m.statusMsg != "" {
		var msgStyle lipgloss.Style
		if m.statusIsError {
			msgStyle = lipgloss.NewStyle().
				Background(ColorPrioCriticalBg).
				Foreground(ColorPrioCritical).
				Bold(true).
				Padding(0, 2)
		} else {
			msgStyle = lipgloss.NewStyle().
				Background(ColorStatusOpenBg).
				Foreground(ColorSuccess).
				Bold(true).
				Padding(0, 2)
		}
		msgSection := msgStyle.Render("✓ " + m.statusMsg)
		remaining := m.width - lipgloss.Width(msgSection)
		if remaining < 0 {
			remaining = 0
		}
		filler := lipgloss.NewStyle().Background(ColorBgDark).Width(remaining).Render("")
		return lipgloss.JoinHorizontal(lipgloss.Bottom, msgSection, filler)
	}

	// ─────────────────────────────────────────────────────────────────────────
	// FILTER BADGE - Current view/filter state + quick hint for label dashboard
	// ─────────────────────────────────────────────────────────────────────────
	var filterTxt string
	var filterIcon string
	if m.focused == focusLabelDashboard {
		filterTxt = "LABELS: j/k nav • h detail • d drilldown • enter filter"
		filterIcon = "🏷️"
	} else if m.showLabelGraphAnalysis && m.labelGraphAnalysisResult != nil {
		filterTxt = fmt.Sprintf("GRAPH %s: esc/q/g close", m.labelGraphAnalysisResult.Label)
		filterIcon = "📊"
	} else if m.showLabelDrilldown && m.labelDrilldownLabel != "" {
		filterTxt = fmt.Sprintf("LABEL %s: enter filter • g graph • esc/q/d close", m.labelDrilldownLabel)
		filterIcon = "🏷️"
	} else {
		switch m.currentFilter {
		case "all":
			filterTxt = "ALL"
			filterIcon = "📋"
		case "open":
			filterTxt = "OPEN"
			filterIcon = "📂"
		case "closed":
			filterTxt = "CLOSED"
			filterIcon = "✅"
		case "ready":
			filterTxt = "READY"
			filterIcon = "🚀"
		default:
			if strings.HasPrefix(m.currentFilter, "recipe:") {
				filterTxt = strings.ToUpper(m.currentFilter[7:])
				filterIcon = "📑"
			} else {
				filterTxt = m.currentFilter
				filterIcon = "🔍"
			}
		}
	}

	filterBadge := lipgloss.NewStyle().
		Background(ColorPrimary).
		Foreground(ColorText).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf("%s %s", filterIcon, filterTxt))

	labelHint := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Background(ColorBgDark).
		Padding(0, 1).
		Render("L:labels • h:detail")

	if m.showAttentionView {
		labelHint = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Background(ColorBgDark).
			Padding(0, 1).
			Render("A:attention • 1-9 filter • esc close")
	}

	// ─────────────────────────────────────────────────────────────────────────
	// STATS SECTION - Issue counts with visual indicators
	// ─────────────────────────────────────────────────────────────────────────
	var statsSection string
	if m.timeTravelMode && m.timeTravelDiff != nil {
		d := m.timeTravelDiff.Summary
		timeTravelStyle := lipgloss.NewStyle().
			Background(ColorPrioHighBg).
			Foreground(ColorWarning).
			Padding(0, 1)
		statsSection = timeTravelStyle.Render(fmt.Sprintf("⏱ %s: +%d ✅%d ~%d",
			m.timeTravelSince, d.IssuesAdded, d.IssuesClosed, d.IssuesModified))
	} else {
		// Polished stats with mini indicators
		statsStyle := lipgloss.NewStyle().
			Background(ColorBgHighlight).
			Foreground(ColorText).
			Padding(0, 1)

		openStyle := lipgloss.NewStyle().Foreground(ColorStatusOpen)
		readyStyle := lipgloss.NewStyle().Foreground(ColorSuccess)
		blockedStyle := lipgloss.NewStyle().Foreground(ColorWarning)
		closedStyle := lipgloss.NewStyle().Foreground(ColorMuted)

		statsContent := fmt.Sprintf("%s%d %s%d %s%d %s%d",
			openStyle.Render("○"),
			m.countOpen,
			readyStyle.Render("◉"),
			m.countReady,
			blockedStyle.Render("◈"),
			m.countBlocked,
			closedStyle.Render("●"),
			m.countClosed)
		statsSection = statsStyle.Render(statsContent)
	}

	// ─────────────────────────────────────────────────────────────────────────
	// UPDATE BADGE - New version available
	// ─────────────────────────────────────────────────────────────────────────
	updateSection := ""
	if m.updateAvailable {
		updateStyle := lipgloss.NewStyle().
			Background(ColorTypeFeature).
			Foreground(ColorBg).
			Bold(true).
			Padding(0, 1)
		updateSection = updateStyle.Render(fmt.Sprintf("⭐ %s", m.updateTag))
	}

	// ─────────────────────────────────────────────────────────────────────────
	// ALERTS BADGE - Project health alerts (bv-168)
	// ─────────────────────────────────────────────────────────────────────────
	alertsSection := ""
	// Count active (non-dismissed) alerts - use simple counters from model
	activeAlerts := len(m.alerts) - len(m.dismissedAlerts)
	if activeAlerts < 0 {
		activeAlerts = 0
	}
	activeCritical := m.alertsCritical
	activeWarning := m.alertsWarning
	if activeAlerts > 0 {
		var alertStyle lipgloss.Style
		var alertIcon string
		if activeCritical > 0 {
			alertStyle = lipgloss.NewStyle().
				Background(ColorPrioCriticalBg).
				Foreground(ColorPrioCritical).
				Bold(true).
				Padding(0, 1)
			alertIcon = "⚠"
		} else if activeWarning > 0 {
			alertStyle = lipgloss.NewStyle().
				Background(ColorPrioHighBg).
				Foreground(ColorWarning).
				Bold(true).
				Padding(0, 1)
			alertIcon = "⚡"
		} else {
			alertStyle = lipgloss.NewStyle().
				Background(ColorBgHighlight).
				Foreground(ColorInfo).
				Padding(0, 1)
			alertIcon = "ℹ"
		}
		alertsSection = alertStyle.Render(fmt.Sprintf("%s %d alerts (!)", alertIcon, activeAlerts))
	}

	// ─────────────────────────────────────────────────────────────────────────
	// WORKSPACE BADGE - Multi-repo mode indicator
	// ─────────────────────────────────────────────────────────────────────────
	workspaceSection := ""
	if m.workspaceMode && m.workspaceSummary != "" {
		workspaceStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#45B7D1")).
			Foreground(ColorBg).
			Bold(true).
			Padding(0, 1)
		workspaceSection = workspaceStyle.Render(fmt.Sprintf("📦 %s", m.workspaceSummary))
	}

	// ─────────────────────────────────────────────────────────────────────────
	// REPO FILTER BADGE - Active repo selection (workspace mode)
	// ─────────────────────────────────────────────────────────────────────────
	repoFilterSection := ""
	if m.workspaceMode && m.activeRepos != nil && len(m.activeRepos) > 0 {
		active := sortedRepoKeys(m.activeRepos)
		label := formatRepoList(active, 3)
		repoStyle := lipgloss.NewStyle().
			Background(ColorBgHighlight).
			Foreground(ColorInfo).
			Bold(true).
			Padding(0, 1)
		repoFilterSection = repoStyle.Render(fmt.Sprintf("🗂 %s", label))
	}

	// ─────────────────────────────────────────────────────────────────────────
	// KEYBOARD HINTS - Context-aware navigation help
	// ─────────────────────────────────────────────────────────────────────────
	keyStyle := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Background(ColorBgSubtle).
		Padding(0, 0)
	sepStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	sep := sepStyle.Render(" │ ")

	var keyHints []string
	if m.showHelp {
		keyHints = append(keyHints, "Press any key to close")
	} else if m.showRecipePicker {
		keyHints = append(keyHints, keyStyle.Render("j/k")+" nav", keyStyle.Render("⏎")+" apply", keyStyle.Render("esc")+" cancel")
	} else if m.showRepoPicker {
		keyHints = append(keyHints, keyStyle.Render("j/k")+" nav", keyStyle.Render("space")+" toggle", keyStyle.Render("⏎")+" apply", keyStyle.Render("esc")+" cancel")
	} else if m.showLabelPicker {
		keyHints = append(keyHints, "type to filter", keyStyle.Render("j/k")+" nav", keyStyle.Render("⏎")+" apply", keyStyle.Render("esc")+" cancel")
	} else if m.focused == focusInsights {
		keyHints = append(keyHints, keyStyle.Render("h/l")+" panels", keyStyle.Render("e")+" explain", keyStyle.Render("⏎")+" jump", keyStyle.Render("?")+" help")
		keyHints = append(keyHints, keyStyle.Render("A")+" attention", keyStyle.Render("F")+" flow")
	} else if m.isGraphView {
		keyHints = append(keyHints, keyStyle.Render("hjkl")+" nav", keyStyle.Render("H/L")+" scroll", keyStyle.Render("⏎")+" view", keyStyle.Render("g")+" list")
	} else if m.isBoardView {
		keyHints = append(keyHints, keyStyle.Render("hjkl")+" nav", keyStyle.Render("G")+" bottom", keyStyle.Render("⏎")+" view", keyStyle.Render("b")+" list")
	} else if m.isActionableView {
		keyHints = append(keyHints, keyStyle.Render("j/k")+" nav", keyStyle.Render("⏎")+" view", keyStyle.Render("a")+" list", keyStyle.Render("?")+" help")
	} else if m.isHistoryView {
		keyHints = append(keyHints, keyStyle.Render("j/k")+" nav", keyStyle.Render("tab")+" focus", keyStyle.Render("⏎")+" jump", keyStyle.Render("H")+" close")
	} else if m.list.FilterState() == list.Filtering {
		mode := "fuzzy"
		if m.semanticSearchEnabled {
			mode = "semantic"
			if m.semanticIndexBuilding {
				mode = "semantic (indexing)"
			}
		}
		keyHints = append(keyHints, keyStyle.Render("esc")+" cancel", keyStyle.Render("ctrl+s")+" "+mode, keyStyle.Render("⏎")+" select")
	} else if m.showTimeTravelPrompt {
		keyHints = append(keyHints, keyStyle.Render("⏎")+" compare", keyStyle.Render("esc")+" cancel")
	} else {
		if m.timeTravelMode {
			keyHints = append(keyHints, keyStyle.Render("t")+" exit diff", keyStyle.Render("C")+" copy", keyStyle.Render("abgi")+" views", keyStyle.Render("?")+" help")
		} else if m.isSplitView {
			keyHints = append(keyHints, keyStyle.Render("tab")+" focus", keyStyle.Render("C")+" copy", keyStyle.Render("E")+" export", keyStyle.Render("?")+" help")
		} else if m.showDetails {
			keyHints = append(keyHints, keyStyle.Render("esc")+" back", keyStyle.Render("C")+" copy", keyStyle.Render("O")+" edit", keyStyle.Render("?")+" help")
		} else {
			keyHints = append(keyHints, keyStyle.Render("⏎")+" details", keyStyle.Render("t")+" diff", keyStyle.Render("S")+" triage", keyStyle.Render("l")+" labels", keyStyle.Render("?")+" help")
			if m.workspaceMode {
				keyHints = append(keyHints, keyStyle.Render("w")+" repos")
			}
		}
	}

	keysSection := lipgloss.NewStyle().
		Foreground(ColorSubtext).
		Padding(0, 1).
		Render(strings.Join(keyHints, sep))

	// ─────────────────────────────────────────────────────────────────────────
	// COUNT BADGE - Total issues displayed
	// ─────────────────────────────────────────────────────────────────────────
	countBadge := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Padding(0, 1).
		Render(fmt.Sprintf("%d issues", len(m.list.Items())))

	// ─────────────────────────────────────────────────────────────────────────
	// ASSEMBLE FOOTER with proper spacing
	// ─────────────────────────────────────────────────────────────────────────
	leftWidth := lipgloss.Width(filterBadge) + lipgloss.Width(labelHint) + lipgloss.Width(statsSection)
	if alertsSection != "" {
		leftWidth += lipgloss.Width(alertsSection) + 1
	}
	if workspaceSection != "" {
		leftWidth += lipgloss.Width(workspaceSection) + 1
	}
	if repoFilterSection != "" {
		leftWidth += lipgloss.Width(repoFilterSection) + 1
	}
	if updateSection != "" {
		leftWidth += lipgloss.Width(updateSection) + 1
	}
	rightWidth := lipgloss.Width(countBadge) + lipgloss.Width(keysSection)

	remaining := m.width - leftWidth - rightWidth - 1
	if remaining < 0 {
		remaining = 0
	}
	filler := lipgloss.NewStyle().Background(ColorBgDark).Width(remaining).Render("")

	// Build the footer
	var parts []string
	parts = append(parts, filterBadge, labelHint)
	if alertsSection != "" {
		parts = append(parts, alertsSection)
	}
	if workspaceSection != "" {
		parts = append(parts, workspaceSection)
	}
	if repoFilterSection != "" {
		parts = append(parts, repoFilterSection)
	}
	if updateSection != "" {
		parts = append(parts, updateSection)
	}
	parts = append(parts, statsSection, filler, countBadge, keysSection)

	return lipgloss.JoinHorizontal(lipgloss.Bottom, parts...)
}
