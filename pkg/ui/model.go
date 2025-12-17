package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Dicklesworthstone/beads_viewer/pkg/analysis"
	"github.com/Dicklesworthstone/beads_viewer/pkg/correlation"
	"github.com/Dicklesworthstone/beads_viewer/pkg/drift"
	"github.com/Dicklesworthstone/beads_viewer/pkg/loader"
	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
	"github.com/Dicklesworthstone/beads_viewer/pkg/recipe"
	"github.com/Dicklesworthstone/beads_viewer/pkg/updater"
	"github.com/Dicklesworthstone/beads_viewer/pkg/watcher"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View width thresholds for adaptive layout
const (
	SplitViewThreshold     = 100
	WideViewThreshold      = 140
	UltraWideViewThreshold = 180
)

// focus represents which UI element has keyboard focus
type focus int

const (
	focusList focus = iota
	focusDetail
	focusBoard
	focusGraph
	focusLabelDashboard
	focusInsights
	focusActionable
	focusRecipePicker
	focusRepoPicker
	focusHelp
	focusQuitConfirm
	focusTimeTravelInput
	focusHistory
	focusAttention
	focusLabelPicker
	focusSprint // Sprint dashboard view (bv-161)
)

// LabelGraphAnalysisResult holds label-specific graph analysis results (bv-109)
type LabelGraphAnalysisResult struct {
	Label        string
	Subgraph     analysis.LabelSubgraph
	PageRank     analysis.LabelPageRankResult
	CriticalPath analysis.LabelCriticalPathResult
}

// UpdateMsg is sent when a new version is available
type UpdateMsg struct {
	TagName string
	URL     string
}

// Phase2ReadyMsg is sent when async graph analysis Phase 2 completes
type Phase2ReadyMsg struct {
	Stats *analysis.GraphStats // The stats that completed, to detect stale messages
}

// WaitForPhase2Cmd returns a command that waits for Phase 2 and sends Phase2ReadyMsg
func WaitForPhase2Cmd(stats *analysis.GraphStats) tea.Cmd {
	return func() tea.Msg {
		stats.WaitForPhase2()
		return Phase2ReadyMsg{Stats: stats}
	}
}

// FileChangedMsg is sent when the beads file changes on disk
type FileChangedMsg struct{}

// WatchFileCmd returns a command that waits for file changes and sends FileChangedMsg
func WatchFileCmd(w *watcher.Watcher) tea.Cmd {
	return func() tea.Msg {
		<-w.Changed()
		return FileChangedMsg{}
	}
}

// CheckUpdateCmd returns a command that checks for updates
func CheckUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		tag, url, err := updater.CheckForUpdates()
		if err == nil && tag != "" {
			return UpdateMsg{TagName: tag, URL: url}
		}
		return nil
	}
}

// HistoryLoadedMsg is sent when background history loading completes
type HistoryLoadedMsg struct {
	Report *correlation.HistoryReport
	Error  error
}

// LoadHistoryCmd returns a command that loads history data in the background
func LoadHistoryCmd(issues []model.Issue, beadsPath string) tea.Cmd {
	return func() tea.Msg {
		var repoPath string
		var err error

		if beadsPath != "" {
			// If beadsPath is provided (single-repo mode), derive repo root from it.
			// Try to resolve absolute path first.
			if absPath, e := filepath.Abs(beadsPath); e == nil {
				dir := filepath.Dir(absPath)
				// Standard layout: <repo_root>/.beads/<file.jsonl>
				if filepath.Base(dir) == ".beads" {
					repoPath = filepath.Dir(dir)
				} else {
					// Legacy/Flat layout: <repo_root>/<file.jsonl>
					repoPath = dir
				}
			}
		}

		// Fallback to CWD if beadsPath is empty (workspace mode) or Abs failed
		if repoPath == "" {
			repoPath, err = os.Getwd()
			if err != nil {
				return HistoryLoadedMsg{Error: err}
			}
		}

		// Convert model.Issue to correlation.BeadInfo
		beads := make([]correlation.BeadInfo, len(issues))
		for i, issue := range issues {
			beads[i] = correlation.BeadInfo{
				ID:     issue.ID,
				Title:  issue.Title,
				Status: string(issue.Status),
			}
		}

		correlator := correlation.NewCorrelator(repoPath, beadsPath)
		opts := correlation.CorrelatorOptions{
			Limit: 500, // Reasonable limit for TUI performance
		}

		report, err := correlator.GenerateReport(beads, opts)
		return HistoryLoadedMsg{Report: report, Error: err}
	}
}

// Model is the main Bubble Tea model for the beads viewer
type Model struct {
	// Data
	issues    []model.Issue
	issueMap  map[string]*model.Issue
	analyzer  *analysis.Analyzer
	analysis  *analysis.GraphStats
	beadsPath string           // Path to beads.jsonl for reloading
	watcher   *watcher.Watcher // File watcher for live reload

	// UI Components
	list               list.Model
	viewport           viewport.Model
	renderer           *MarkdownRenderer
	board              BoardModel
	labelDashboard     LabelDashboardModel
	velocityComparison VelocityComparisonModel // bv-125
	shortcutsSidebar   ShortcutsSidebar        // bv-3qi5
	graphView          GraphModel
	insightsPanel      InsightsModel
	theme              Theme

	// Update State
	updateAvailable bool
	updateTag       string
	updateURL       string

	// Focus and View State
	focused                  focus
	isSplitView              bool
	isBoardView              bool
	isGraphView              bool
	isActionableView         bool
	isHistoryView            bool
	showDetails              bool
	showHelp                 bool
	helpScroll               int // Scroll offset for help overlay
	showQuitConfirm          bool
	ready                    bool
	width                    int
	height                   int
	showLabelHealthDetail    bool
	showLabelDrilldown       bool
	labelHealthDetail        *analysis.LabelHealth
	labelHealthDetailFlow    labelFlowSummary
	labelDrilldownLabel      string
	labelDrilldownIssues     []model.Issue
	labelDrilldownCache      map[string][]model.Issue
	showLabelGraphAnalysis   bool
	labelGraphAnalysisResult *LabelGraphAnalysisResult
	showAttentionView        bool
	showShortcutsSidebar     bool // bv-3qi5 toggleable shortcuts sidebar
	labelHealthCached        bool
	labelHealthCache         analysis.LabelAnalysisResult
	attentionCached          bool
	attentionCache           analysis.LabelAttentionResult
	flowMatrixText           string

	// Actionable view
	actionableView ActionableModel

	// History view
	historyView       HistoryModel
	historyLoading    bool // True while history is being loaded in background
	historyLoadFailed bool // True if history loading failed

	// Filter state
	currentFilter         string
	semanticSearchEnabled bool
	semanticIndexBuilding bool
	semanticSearch        *SemanticSearch

	// Stats (cached)
	countOpen    int
	countReady   int
	countBlocked int
	countClosed  int

	// Priority hints
	showPriorityHints bool
	priorityHints     map[string]*analysis.PriorityRecommendation // issueID -> recommendation

	// Triage insights (bv-151)
	triageScores  map[string]float64                // issueID -> triage score
	triageReasons map[string]analysis.TriageReasons // issueID -> reasons
	unblocksMap   map[string][]string               // issueID -> IDs that would be unblocked
	quickWinSet   map[string]bool                   // issueID -> true if quick win
	blockerSet    map[string]bool                   // issueID -> true if significant blocker

	// Recipe picker
	showRecipePicker bool
	recipePicker     RecipePickerModel
	activeRecipe     *recipe.Recipe
	recipeLoader     *recipe.Loader

	// Label picker (bv-126)
	showLabelPicker bool
	labelPicker     LabelPickerModel

	// Repo picker (workspace mode)
	showRepoPicker bool
	repoPicker     RepoPickerModel

	// Time-travel mode
	timeTravelMode   bool
	timeTravelDiff   *analysis.SnapshotDiff
	timeTravelSince  string
	newIssueIDs      map[string]bool // Issues in diff.NewIssues
	closedIssueIDs   map[string]bool // Issues in diff.ClosedIssues
	modifiedIssueIDs map[string]bool // Issues in diff.ModifiedIssues

	// Time-travel input prompt
	timeTravelInput      textinput.Model
	showTimeTravelPrompt bool

	// Status message (for temporary feedback)
	statusMsg     string
	statusIsError bool

	// Workspace mode state
	workspaceMode    bool            // True when viewing multiple repos
	availableRepos   []string        // List of repo prefixes available
	activeRepos      map[string]bool // Which repos are currently shown (nil = all)
	workspaceSummary string          // Summary text for footer (e.g., "3 repos")

	// Alerts panel (bv-168)
	alerts          []drift.Alert
	alertsCritical  int
	alertsWarning   int
	alertsInfo      int
	showAlertsPanel bool
	alertsCursor    int
	dismissedAlerts map[string]bool

	// Sprint view (bv-161)
	sprints        []model.Sprint
	selectedSprint *model.Sprint
	isSprintView   bool
	sprintViewText string
}

// NewModel creates a new Model from the given issues
// beadsPath is the path to the beads.jsonl file for live reload support
func NewModel(issues []model.Issue, activeRecipe *recipe.Recipe, beadsPath string) Model {
	// Graph Analysis - Phase 1 is instant, Phase 2 runs in background
	analyzer := analysis.NewAnalyzer(issues)
	graphStats := analyzer.AnalyzeAsync(context.Background())

	// Sort issues
	if activeRecipe != nil && activeRecipe.Sort.Field != "" {
		r := activeRecipe
		descending := r.Sort.Direction == "desc"

		sort.Slice(issues, func(i, j int) bool {
			less := false
			switch r.Sort.Field {
			case "priority":
				less = issues[i].Priority < issues[j].Priority
			case "created", "created_at":
				less = issues[i].CreatedAt.Before(issues[j].CreatedAt)
			case "updated", "updated_at":
				less = issues[i].UpdatedAt.Before(issues[j].UpdatedAt)
			case "impact":
				less = graphStats.GetCriticalPathScore(issues[i].ID) < graphStats.GetCriticalPathScore(issues[j].ID)
			case "pagerank":
				less = graphStats.GetPageRankScore(issues[i].ID) < graphStats.GetPageRankScore(issues[j].ID)
			default:
				less = issues[i].Priority < issues[j].Priority
			}
			if descending {
				return !less
			}
			return less
		})
	} else {
		// Default Sort: Open first, then by Priority (ascending), then by date (newest first)
		sort.Slice(issues, func(i, j int) bool {
			iClosed := issues[i].Status == model.StatusClosed
			jClosed := issues[j].Status == model.StatusClosed
			if iClosed != jClosed {
				return !iClosed // Open issues first
			}
			if issues[i].Priority != issues[j].Priority {
				return issues[i].Priority < issues[j].Priority // Lower priority number = higher priority
			}
			return issues[i].CreatedAt.After(issues[j].CreatedAt) // Newer first
		})
	}

	// Build lookup map
	issueMap := make(map[string]*model.Issue, len(issues))

	// Build list items - scores may be 0 until Phase 2 completes
	items := make([]list.Item, len(issues))
	for i := range issues {
		issueMap[issues[i].ID] = &issues[i]

		items[i] = IssueItem{
			Issue:      issues[i],
			GraphScore: graphStats.GetPageRankScore(issues[i].ID),
			Impact:     graphStats.GetCriticalPathScore(issues[i].ID),
			RepoPrefix: ExtractRepoPrefix(issues[i].ID),
		}
	}

	// Compute stats
	cOpen, cReady, cBlocked, cClosed := 0, 0, 0, 0
	for i := range issues {
		issue := &issues[i]
		if issue.Status == model.StatusClosed {
			cClosed++
			continue
		}

		cOpen++
		if issue.Status == model.StatusBlocked {
			cBlocked++
			continue
		}

		// Check if blocked by open dependencies
		isBlocked := false
		for _, dep := range issue.Dependencies {
			if dep == nil || !dep.Type.IsBlocking() {
				continue
			}
			if blocker, exists := issueMap[dep.DependsOnID]; exists && blocker.Status != model.StatusClosed {
				isBlocked = true
				break
			}
		}
		if !isBlocked {
			cReady++
		}
	}

	// Theme
	theme := DefaultTheme(lipgloss.NewRenderer(os.Stdout))

	// List setup
	delegate := IssueDelegate{Theme: theme, WorkspaceMode: false}
	l := list.New(items, delegate, 0, 0)
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	// Clear all default styles that might add extra lines
	l.Styles.Title = lipgloss.NewStyle()
	l.Styles.TitleBar = lipgloss.NewStyle()
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(theme.Primary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(theme.Primary)
	l.Styles.StatusBar = lipgloss.NewStyle()
	l.Styles.StatusEmpty = lipgloss.NewStyle()
	l.Styles.StatusBarActiveFilter = lipgloss.NewStyle()
	l.Styles.StatusBarFilterCount = lipgloss.NewStyle()
	l.Styles.NoItems = lipgloss.NewStyle()
	l.Styles.PaginationStyle = lipgloss.NewStyle()
	l.Styles.HelpStyle = lipgloss.NewStyle()

	// Theme-aware markdown renderer
	renderer := NewMarkdownRendererWithTheme(80, theme)

	// Initialize sub-components
	board := NewBoardModel(issues, theme)
	labelDashboard := NewLabelDashboardModel(theme)
	velocityComparison := NewVelocityComparisonModel(theme) // bv-125
	shortcutsSidebar := NewShortcutsSidebar(theme)          // bv-3qi5
	ins := graphStats.GenerateInsights(len(issues))         // allow UI to show as many as fit
	insightsPanel := NewInsightsModel(ins, issueMap, theme)
	graphView := NewGraphModel(issues, &ins, theme)

	// Priority hints are generated asynchronously when Phase 2 completes
	// This avoids blocking startup on expensive graph analysis
	priorityHints := make(map[string]*analysis.PriorityRecommendation)

	// Compute triage insights (bv-151)
	triageResult := analysis.ComputeTriage(issues)
	triageScores := make(map[string]float64, len(triageResult.Recommendations))
	triageReasons := make(map[string]analysis.TriageReasons, len(triageResult.Recommendations))
	quickWinSet := make(map[string]bool, len(triageResult.QuickWins))
	blockerSet := make(map[string]bool, len(triageResult.BlockersToClear))
	unblocksMap := make(map[string][]string, len(triageResult.Recommendations))

	for _, rec := range triageResult.Recommendations {
		triageScores[rec.ID] = rec.Score
		if len(rec.Reasons) > 0 {
			triageReasons[rec.ID] = analysis.TriageReasons{
				Primary:    rec.Reasons[0],
				All:        rec.Reasons,
				ActionHint: rec.Action,
			}
		}
		unblocksMap[rec.ID] = rec.UnblocksIDs
	}
	for _, qw := range triageResult.QuickWins {
		quickWinSet[qw.ID] = true
	}
	for _, bl := range triageResult.BlockersToClear {
		blockerSet[bl.ID] = true
	}

	// Update items with triage data
	for i := range items {
		if issueItem, ok := items[i].(IssueItem); ok {
			issueItem.TriageScore = triageScores[issueItem.Issue.ID]
			if reasons, exists := triageReasons[issueItem.Issue.ID]; exists {
				issueItem.TriageReason = reasons.Primary
				issueItem.TriageReasons = reasons.All
			}
			issueItem.IsQuickWin = quickWinSet[issueItem.Issue.ID]
			issueItem.IsBlocker = blockerSet[issueItem.Issue.ID]
			issueItem.UnblocksCount = len(unblocksMap[issueItem.Issue.ID])
			items[i] = issueItem
		}
	}

	// Initialize recipe loader
	recipeLoader := recipe.NewLoader()
	_ = recipeLoader.Load() // Load recipes (errors are non-fatal, will just show empty)
	recipePicker := NewRecipePickerModel(recipeLoader.List(), theme)

	// Initialize label picker (bv-126)
	labelExtraction := analysis.ExtractLabels(issues)
	labelPicker := NewLabelPickerModel(labelExtraction.Labels, theme)

	// Initialize time-travel input
	ti := textinput.New()
	ti.Placeholder = "HEAD~5, main, v1.0.0, 2024-01-01..."
	ti.CharLimit = 100
	ti.Width = 40
	ti.Prompt = "⏱️  Revision: "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Base.GetForeground())

	// Initialize file watcher for live reload
	var fileWatcher *watcher.Watcher
	var watcherErr error
	if beadsPath != "" {
		w, err := watcher.NewWatcher(beadsPath,
			watcher.WithDebounceDuration(200*time.Millisecond),
		)
		if err != nil {
			watcherErr = err
		} else if err := w.Start(); err != nil {
			watcherErr = err
		} else {
			fileWatcher = w
		}
	}

	// Semantic search (bv-9gf.3): initialized lazily on first toggle.
	semanticSearch := NewSemanticSearch()
	semanticIDs := make([]string, 0, len(items))
	for _, it := range items {
		if issueItem, ok := it.(IssueItem); ok {
			semanticIDs = append(semanticIDs, issueItem.Issue.ID)
		}
	}
	semanticSearch.SetIDs(semanticIDs)

	// Build initial status message if watcher failed
	var initialStatus string
	var initialStatusErr bool
	if watcherErr != nil {
		initialStatus = fmt.Sprintf("Live reload unavailable: %v", watcherErr)
		initialStatusErr = true
	}

	// Precompute drift/health alerts (bv-168)
	alerts, alertsCritical, alertsWarning, alertsInfo := computeAlerts(issues, graphStats, analyzer)

	// Load sprints from the same directory as beadsPath (bv-161)
	var sprints []model.Sprint
	if beadsPath != "" {
		beadsDir := filepath.Dir(beadsPath)
		if loaded, err := loader.LoadSprintsFromFile(filepath.Join(beadsDir, loader.SprintsFileName)); err == nil {
			sprints = loaded
		}
	}

	return Model{
		issues:              issues,
		issueMap:            issueMap,
		analyzer:            analyzer,
		analysis:            graphStats,
		beadsPath:           beadsPath,
		watcher:             fileWatcher,
		list:                l,
		renderer:            renderer,
		board:               board,
		labelDashboard:      labelDashboard,
		velocityComparison:  velocityComparison,
		shortcutsSidebar:    shortcutsSidebar,
		graphView:           graphView,
		insightsPanel:       insightsPanel,
		theme:               theme,
		currentFilter:       "all",
		semanticSearch:      semanticSearch,
		focused:             focusList,
		countOpen:           cOpen,
		countReady:          cReady,
		countBlocked:        cBlocked,
		countClosed:         cClosed,
		priorityHints:       priorityHints,
		showPriorityHints:   false, // Off by default, toggle with 'p'
		triageScores:        triageScores,
		triageReasons:       triageReasons,
		unblocksMap:         unblocksMap,
		quickWinSet:         quickWinSet,
		blockerSet:          blockerSet,
		recipeLoader:        recipeLoader,
		recipePicker:        recipePicker,
		activeRecipe:        activeRecipe,
		labelPicker:         labelPicker,
		labelDrilldownCache: make(map[string][]model.Issue),
		timeTravelInput:     ti,
		statusMsg:           initialStatus,
		statusIsError:       initialStatusErr,
		historyLoading:      len(issues) > 0, // Will be loaded in Init()
		// Alerts panel (bv-168)
		alerts:          alerts,
		alertsCritical:  alertsCritical,
		alertsWarning:   alertsWarning,
		alertsInfo:      alertsInfo,
		dismissedAlerts: make(map[string]bool),
		// Sprint view (bv-161)
		sprints: sprints,
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{CheckUpdateCmd(), WaitForPhase2Cmd(m.analysis)}
	if m.watcher != nil {
		cmds = append(cmds, WatchFileCmd(m.watcher))
	}
	// Start loading history in background
	if len(m.issues) > 0 {
		cmds = append(cmds, LoadHistoryCmd(m.issues, m.beadsPath))
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case UpdateMsg:
		m.updateAvailable = true
		m.updateTag = msg.TagName
		m.updateURL = msg.URL

	case SemanticIndexReadyMsg:
		m.semanticIndexBuilding = false
		if msg.Error != nil {
			// If indexing fails, revert to fuzzy mode for predictable behavior.
			m.semanticSearchEnabled = false
			m.list.Filter = list.DefaultFilter
			m.statusMsg = fmt.Sprintf("Semantic search unavailable: %v", msg.Error)
			m.statusIsError = true
			break
		}
		if m.semanticSearch != nil {
			m.semanticSearch.SetIndex(msg.Index, msg.Embedder)
		}
		if !msg.Loaded {
			m.statusMsg = fmt.Sprintf("Semantic index built (%d embedded)", msg.Stats.Embedded)
		} else if msg.Stats.Changed() {
			m.statusMsg = fmt.Sprintf("Semantic index updated (+%d ~%d -%d)", msg.Stats.Added, msg.Stats.Updated, msg.Stats.Removed)
		} else {
			m.statusMsg = "Semantic index up to date"
		}
		m.statusIsError = false

		// Refresh current filter view if the user is actively searching.
		if m.semanticSearchEnabled && m.list.FilterState() != list.Unfiltered {
			prevState := m.list.FilterState()
			filterText := m.list.FilterInput.Value()
			m.list.SetFilterText(filterText)
			if prevState == list.Filtering {
				m.list.SetFilterState(list.Filtering)
			}
		}

	case Phase2ReadyMsg:
		// Ignore stale Phase2 completions (from before a file reload)
		if msg.Stats != m.analysis {
			return m, nil
		}
		// Phase 2 analysis complete - regenerate insights with full data
		ins := m.analysis.GenerateInsights(len(m.issues))
		m.insightsPanel = NewInsightsModel(ins, m.issueMap, m.theme)
		bodyHeight := m.height - 1
		if bodyHeight < 5 {
			bodyHeight = 5
		}
		m.insightsPanel.SetSize(m.width, bodyHeight)
		m.graphView.SetIssues(m.issues, &ins)

		// Generate triage for priority panel (bv-91)
		triage := analysis.ComputeTriage(m.issues)
		m.insightsPanel.SetTopPicks(triage.QuickRef.TopPicks)

		// Set full recommendations with breakdown for priority radar (bv-93)
		dataHash := fmt.Sprintf("v%s@%s#%d", triage.Meta.Version, triage.Meta.GeneratedAt.Format("15:04:05"), triage.Meta.IssueCount)
		m.insightsPanel.SetRecommendations(triage.Recommendations, dataHash)

		// Generate priority recommendations now that Phase 2 is ready
		recommendations := m.analyzer.GenerateRecommendations()
		m.priorityHints = make(map[string]*analysis.PriorityRecommendation, len(recommendations))
		for i := range recommendations {
			m.priorityHints[recommendations[i].IssueID] = &recommendations[i]
		}

		// Refresh alerts now that full Phase 2 metrics (cycles, etc.) are available
		m.alerts, m.alertsCritical, m.alertsWarning, m.alertsInfo = computeAlerts(m.issues, m.analysis, m.analyzer)

		// Invalidate label health cache since we have new graph metrics (criticality)
		m.labelHealthCached = false
		if m.focused == focusLabelDashboard {
			cfg := analysis.DefaultLabelHealthConfig()
			m.labelHealthCache = analysis.ComputeAllLabelHealth(m.issues, cfg, time.Now().UTC(), m.analysis)
			m.labelHealthCached = true
			m.labelDashboard.SetData(m.labelHealthCache.Labels)
			m.statusMsg = fmt.Sprintf("Labels: %d total • critical %d • warning %d", m.labelHealthCache.TotalLabels, m.labelHealthCache.CriticalCount, m.labelHealthCache.WarningCount)
		}

		// Re-sort issues if sorting by Phase 2 metrics (impact/pagerank)
		if m.activeRecipe != nil {
			switch m.activeRecipe.Sort.Field {
			case "impact", "pagerank":
				descending := m.activeRecipe.Sort.Direction == "desc"
				sort.Slice(m.issues, func(i, j int) bool {
					var less bool
					if m.activeRecipe.Sort.Field == "impact" {
						less = m.analysis.GetCriticalPathScore(m.issues[i].ID) < m.analysis.GetCriticalPathScore(m.issues[j].ID)
					} else {
						less = m.analysis.GetPageRankScore(m.issues[i].ID) < m.analysis.GetPageRankScore(m.issues[j].ID)
					}
					if descending {
						return !less
					}
					return less
				})
				// Rebuild issueMap after re-sort (pointers become stale after sorting)
				for i := range m.issues {
					m.issueMap[m.issues[i].ID] = &m.issues[i]
				}
			}
		}

		// Re-apply recipe filter if active (to update scores while preserving filter)
		// Otherwise, update list respecting current filter (open/ready/etc.)
		if m.activeRecipe != nil {
			m.applyRecipe(m.activeRecipe)
		} else {
			m.applyFilter()
		}

	case HistoryLoadedMsg:
		// Background history loading completed
		m.historyLoading = false
		if msg.Error != nil {
			m.historyLoadFailed = true
			m.statusMsg = fmt.Sprintf("History load failed: %v", msg.Error)
			m.statusIsError = true
		} else if msg.Report != nil {
			m.historyView = NewHistoryModel(msg.Report, m.theme)
			m.historyView.SetSize(m.width, m.height-1)
			// Refresh detail pane if visible
			if m.isSplitView || m.showDetails {
				m.updateViewportContent()
			}
		}

	case FileChangedMsg:
		// File changed on disk - reload issues and recompute analysis
		if m.beadsPath == "" {
			// Re-start watch for next change
			if m.watcher != nil {
				cmds = append(cmds, WatchFileCmd(m.watcher))
			}
			return m, tea.Batch(cmds...)
		}

		// Clear ephemeral overlays tied to old data
		m.clearAttentionOverlay()

		// Exit time-travel mode if active (file changed, show current state)
		if m.timeTravelMode {
			m.timeTravelMode = false
			m.timeTravelDiff = nil
			m.timeTravelSince = ""
			m.newIssueIDs = nil
			m.closedIssueIDs = nil
			m.modifiedIssueIDs = nil
		}

		// Reload issues from disk
		// Use custom warning handler to prevent stderr pollution during TUI render (bv-fix)
		var reloadWarnings []string
		newIssues, err := loader.LoadIssuesFromFileWithOptions(m.beadsPath, loader.ParseOptions{
			WarningHandler: func(msg string) {
				reloadWarnings = append(reloadWarnings, msg)
			},
		})
		if err != nil {
			m.statusMsg = fmt.Sprintf("Reload error: %v", err)
			m.statusIsError = true
			// Re-start watch for next change
			if m.watcher != nil {
				cmds = append(cmds, WatchFileCmd(m.watcher))
			}
			return m, tea.Batch(cmds...)
		}

		// Store selected issue ID to restore position after reload
		var selectedID string
		if sel := m.list.SelectedItem(); sel != nil {
			if item, ok := sel.(IssueItem); ok {
				selectedID = item.Issue.ID
			}
		}

		// Apply default sorting (Open first, Priority, Date)
		sort.Slice(newIssues, func(i, j int) bool {
			iClosed := newIssues[i].Status == model.StatusClosed
			jClosed := newIssues[j].Status == model.StatusClosed
			if iClosed != jClosed {
				return !iClosed
			}
			if newIssues[i].Priority != newIssues[j].Priority {
				return newIssues[i].Priority < newIssues[j].Priority
			}
			return newIssues[i].CreatedAt.After(newIssues[j].CreatedAt)
		})

		// Recompute analysis (async Phase 1/Phase 2) with caching
		m.issues = newIssues
		cachedAnalyzer := analysis.NewCachedAnalyzer(newIssues, nil)
		m.analyzer = cachedAnalyzer.Analyzer
		m.analysis = cachedAnalyzer.AnalyzeAsync(context.Background())
		cacheHit := cachedAnalyzer.WasCacheHit()
		m.labelHealthCached = false
		m.attentionCached = false
		m.flowMatrixText = ""

		// Rebuild lookup map
		m.issueMap = make(map[string]*model.Issue, len(newIssues))
		for i := range m.issues {
			m.issueMap[m.issues[i].ID] = &m.issues[i]
		}

		// Clear stale priority hints (will be repopulated after Phase 2)
		m.priorityHints = make(map[string]*analysis.PriorityRecommendation)

		// Recompute stats
		m.countOpen, m.countReady, m.countBlocked, m.countClosed = 0, 0, 0, 0
		for i := range m.issues {
			issue := &m.issues[i]
			if issue.Status == model.StatusClosed {
				m.countClosed++
				continue
			}
			m.countOpen++
			if issue.Status == model.StatusBlocked {
				m.countBlocked++
				continue
			}
			isBlocked := false
			for _, dep := range issue.Dependencies {
				if dep == nil || !dep.Type.IsBlocking() {
					continue
				}
				if blocker, exists := m.issueMap[dep.DependsOnID]; exists && blocker.Status != model.StatusClosed {
					isBlocked = true
					break
				}
			}
			if !isBlocked {
				m.countReady++
			}
		}

		// Recompute alerts for refreshed dataset
		m.alerts, m.alertsCritical, m.alertsWarning, m.alertsInfo = computeAlerts(m.issues, m.analysis, m.analyzer)
		m.dismissedAlerts = make(map[string]bool)
		m.showAlertsPanel = false

		// Rebuild list items
		items := make([]list.Item, len(m.issues))
		for i := range m.issues {
			items[i] = IssueItem{
				Issue:      m.issues[i],
				GraphScore: m.analysis.GetPageRankScore(m.issues[i].ID),
				Impact:     m.analysis.GetCriticalPathScore(m.issues[i].ID),
				RepoPrefix: ExtractRepoPrefix(m.issues[i].ID),
			}
		}
		m.list.SetItems(items)
		m.updateSemanticIDs(items)

		// Restore selection position
		if selectedID != "" {
			for i, item := range m.list.Items() {
				if issueItem, ok := item.(IssueItem); ok && issueItem.Issue.ID == selectedID {
					m.list.Select(i)
					break
				}
			}
		}

		// Regenerate sub-views (with Phase 1 data; Phase 2 will update via Phase2ReadyMsg)
		ins := m.analysis.GenerateInsights(len(m.issues))
		m.insightsPanel = NewInsightsModel(ins, m.issueMap, m.theme)
		bodyHeight := m.height - 1
		if bodyHeight < 5 {
			bodyHeight = 5
		}
		m.insightsPanel.SetSize(m.width, bodyHeight)
		m.graphView.SetIssues(m.issues, &ins)

		// Generate priority recommendations now that Phase 2 is ready
		m.board = NewBoardModel(m.issues, m.theme)

		// Re-apply recipe filter if active
		if m.activeRecipe != nil {
			m.applyRecipe(m.activeRecipe)
		}

		// Reload sprints (bv-161)
		if m.beadsPath != "" {
			beadsDir := filepath.Dir(m.beadsPath)
			if loaded, err := loader.LoadSprintsFromFile(filepath.Join(beadsDir, loader.SprintsFileName)); err == nil {
				m.sprints = loaded
				// If we have a selected sprint, try to refresh it
				if m.selectedSprint != nil {
					found := false
					for i := range m.sprints {
						if m.sprints[i].ID == m.selectedSprint.ID {
							m.selectedSprint = &m.sprints[i]
							m.sprintViewText = m.renderSprintDashboard()
							found = true
							break
						}
					}
					if !found {
						m.selectedSprint = nil
						m.sprintViewText = "Sprint not found"
					}
				}
			}
		}

		// Keep semantic index current when enabled.
		if m.semanticSearchEnabled && !m.semanticIndexBuilding {
			m.semanticIndexBuilding = true
			cmds = append(cmds, BuildSemanticIndexCmd(m.issues))
		}

		if cacheHit {
			m.statusMsg = fmt.Sprintf("Reloaded %d issues (cached)", len(newIssues))
		} else {
			m.statusMsg = fmt.Sprintf("Reloaded %d issues", len(newIssues))
		}
		if len(reloadWarnings) > 0 {
			m.statusMsg += fmt.Sprintf(" (%d warnings)", len(reloadWarnings))
		}
		m.statusIsError = false
		// Invalidate label-derived caches
		m.labelHealthCached = false
		m.labelDrilldownCache = make(map[string][]model.Issue)
		m.updateViewportContent()

		// Re-start watching for next change + wait for Phase 2
		if m.watcher != nil {
			cmds = append(cmds, WatchFileCmd(m.watcher))
		}
		cmds = append(cmds, WaitForPhase2Cmd(m.analysis))
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Clear status message on any keypress
		m.statusMsg = ""
		m.statusIsError = false

		// Close label health detail modal if open
		if m.showLabelHealthDetail {
			s := msg.String()
			if s == "esc" || s == "q" || s == "enter" || s == "h" {
				m.showLabelHealthDetail = false
				m.labelHealthDetail = nil
				return m, nil
			}
			if s == "d" && m.labelHealthDetail != nil {
				// open drilldown from detail modal
				m.labelDrilldownLabel = m.labelHealthDetail.Label
				m.labelDrilldownIssues = m.filterIssuesByLabel(m.labelDrilldownLabel)
				m.showLabelDrilldown = true
				m.showLabelHealthDetail = false
				return m, nil
			}
		}

		// Handle label drilldown modal if open
		if m.showLabelDrilldown {
			s := msg.String()
			switch s {
			case "enter":
				// Apply label filter to main list and close drilldown
				if m.labelDrilldownLabel != "" {
					m.currentFilter = "label:" + m.labelDrilldownLabel
					m.applyFilter()
					m.focused = focusList
				}
				m.showLabelDrilldown = false
				m.labelDrilldownLabel = ""
				m.labelDrilldownIssues = nil
				return m, nil
			case "g":
				// Show graph analysis sub-view (bv-109)
				if m.labelDrilldownLabel != "" {
					sg := analysis.ComputeLabelSubgraph(m.issues, m.labelDrilldownLabel)
					pr := analysis.ComputeLabelPageRank(sg)
					cp := analysis.ComputeLabelCriticalPath(sg)
					m.labelGraphAnalysisResult = &LabelGraphAnalysisResult{
						Label:        m.labelDrilldownLabel,
						Subgraph:     sg,
						PageRank:     pr,
						CriticalPath: cp,
					}
					m.showLabelGraphAnalysis = true
				}
				return m, nil
			case "esc", "q", "d":
				m.showLabelDrilldown = false
				m.labelDrilldownLabel = ""
				m.labelDrilldownIssues = nil
				return m, nil
			}
		}

		// Handle label graph analysis sub-view (bv-109)
		if m.showLabelGraphAnalysis {
			s := msg.String()
			switch s {
			case "esc", "q", "g":
				m.showLabelGraphAnalysis = false
				m.labelGraphAnalysisResult = nil
				return m, nil
			}
		}

		// Handle attention view quick jumps (bv-117)
		if m.showAttentionView {
			s := msg.String()
			switch {
			case s == "esc" || s == "q" || s == "d":
				m.showAttentionView = false
				m.insightsPanel.extraText = ""
				return m, nil
			case len(s) == 1 && s[0] >= '1' && s[0] <= '9':
				if len(m.attentionCache.Labels) == 0 {
					return m, nil
				}
				idx := int(s[0] - '1')
				if idx >= 0 && idx < len(m.attentionCache.Labels) {
					label := m.attentionCache.Labels[idx].Label
					m.currentFilter = "label:" + label
					m.applyFilter()
					m.statusMsg = fmt.Sprintf("Filtered to label %s (attention #%d)", label, idx+1)
					m.statusIsError = false
				}
				return m, nil
			}
		}

		// Handle alerts panel modal if open (bv-168)
		if m.showAlertsPanel {
			// Build list of active (non-dismissed) alerts
			var activeAlerts []drift.Alert
			for _, a := range m.alerts {
				if !m.dismissedAlerts[alertKey(a)] {
					activeAlerts = append(activeAlerts, a)
				}
			}
			s := msg.String()
			switch s {
			case "j", "down":
				if m.alertsCursor < len(activeAlerts)-1 {
					m.alertsCursor++
				}
				return m, nil
			case "k", "up":
				if m.alertsCursor > 0 {
					m.alertsCursor--
				}
				return m, nil
			case "enter":
				// Jump to the issue referenced by the selected alert
				if m.alertsCursor < len(activeAlerts) {
					issueID := activeAlerts[m.alertsCursor].IssueID
					if issueID != "" {
						// Find the issue in the list and select it
						for i, item := range m.list.Items() {
							if it, ok := item.(IssueItem); ok && it.Issue.ID == issueID {
								m.list.Select(i)
								break
							}
						}
					}
				}
				m.showAlertsPanel = false
				return m, nil
			case "d":
				// Dismiss the selected alert
				if m.alertsCursor < len(activeAlerts) {
					key := alertKey(activeAlerts[m.alertsCursor])
					m.dismissedAlerts[key] = true
					// Adjust cursor if needed
					remaining := 0
					for _, a := range m.alerts {
						if !m.dismissedAlerts[alertKey(a)] {
							remaining++
						}
					}
					if m.alertsCursor >= remaining {
						m.alertsCursor = remaining - 1
					}
					if m.alertsCursor < 0 {
						m.alertsCursor = 0
					}
					// Close panel if no alerts left
					if remaining == 0 {
						m.showAlertsPanel = false
					}
				}
				return m, nil
			case "esc", "q", "!":
				m.showAlertsPanel = false
				return m, nil
			}
			return m, nil
		}

		// Handle repo picker overlay (workspace mode) before global keys (esc/q/etc.)
		if m.showRepoPicker {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			m = m.handleRepoPickerKeys(msg)
			return m, nil
		}

		// Handle recipe picker overlay before global keys (esc/q/etc.)
		if m.showRecipePicker {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			m = m.handleRecipePickerKeys(msg)
			return m, nil
		}

		// Handle quit confirmation first
		if m.showQuitConfirm {
			switch msg.String() {
			case "esc", "y", "Y":
				return m, tea.Quit
			default:
				m.showQuitConfirm = false
				m.focused = focusList
				return m, nil
			}
		}

		// Handle help overlay toggle (? or F1)
		if (msg.String() == "?" || msg.String() == "f1") && m.list.FilterState() != list.Filtering {
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.focused = focusHelp
				m.helpScroll = 0 // Reset scroll position when opening help
			} else {
				m.focused = focusList
			}
			return m, nil
		}

		// Handle shortcuts sidebar toggle (F2) - bv-3qi5
		if msg.String() == "f2" && m.list.FilterState() != list.Filtering {
			m.showShortcutsSidebar = !m.showShortcutsSidebar
			if m.showShortcutsSidebar {
				m.shortcutsSidebar.ResetScroll()
				m.statusMsg = "Shortcuts sidebar: F2 hide | ctrl+j/k scroll"
				m.statusIsError = false
			} else {
				m.statusMsg = ""
			}
			return m, nil
		}

		// Handle shortcuts sidebar scrolling (Ctrl+j/k when sidebar visible) - bv-3qi5
		if m.showShortcutsSidebar && m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "ctrl+j":
				m.shortcutsSidebar.ScrollDown()
				return m, nil
			case "ctrl+k":
				m.shortcutsSidebar.ScrollUp()
				return m, nil
			}
		}

		// Semantic search toggle (bv-9gf.3)
		if msg.String() == "ctrl+s" && m.focused == focusList {
			m.statusIsError = false
			m.semanticSearchEnabled = !m.semanticSearchEnabled
			if m.semanticSearchEnabled {
				if m.semanticSearch != nil {
					m.list.Filter = m.semanticSearch.Filter
					if !m.semanticSearch.Snapshot().Ready && !m.semanticIndexBuilding {
						m.semanticIndexBuilding = true
						m.statusMsg = "Semantic search: building index…"
						cmds = append(cmds, BuildSemanticIndexCmd(m.issues))
					} else if !m.semanticSearch.Snapshot().Ready && m.semanticIndexBuilding {
						m.statusMsg = "Semantic search: indexing…"
					} else {
						m.statusMsg = "Semantic search enabled"
					}
				} else {
					m.semanticSearchEnabled = false
					m.list.Filter = list.DefaultFilter
					m.statusMsg = "Semantic search unavailable"
					m.statusIsError = true
				}
			} else {
				m.list.Filter = list.DefaultFilter
				m.statusMsg = "Fuzzy search enabled"
			}

			// Refresh the current list filter results immediately.
			prevState := m.list.FilterState()
			filterText := m.list.FilterInput.Value()
			if prevState != list.Unfiltered {
				m.list.SetFilterText(filterText)
				if prevState == list.Filtering {
					m.list.SetFilterState(list.Filtering)
				}
			}

			return m, tea.Batch(cmds...)
		}

		// If help is showing, handle navigation keys for scrolling
		if m.focused == focusHelp {
			m = m.handleHelpKeys(msg)
			return m, nil
		}

		// Handle time-travel input first (before global keys intercept letters)
		// But allow ctrl+c to always quit
		if m.focused == focusTimeTravelInput {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			m = m.handleTimeTravelInputKeys(msg)
			return m, nil
		}

		// Handle keys when not filtering
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit

			case "q":
				// q closes current view or quits if at top level
				if m.showDetails && !m.isSplitView {
					m.showDetails = false
					m.focused = focusList
					return m, nil
				}
				if m.focused == focusInsights {
					m.focused = focusList
					return m, nil
				}
				if m.isGraphView {
					m.isGraphView = false
					m.focused = focusList
					return m, nil
				}
				if m.isBoardView {
					m.isBoardView = false
					m.focused = focusList
					return m, nil
				}
				return m, tea.Quit

			case "esc":
				// Escape closes modals and goes back
				if m.showDetails && !m.isSplitView {
					m.showDetails = false
					m.focused = focusList
					return m, nil
				}
				if m.focused == focusInsights {
					m.focused = focusList
					return m, nil
				}
				if m.isGraphView {
					m.isGraphView = false
					m.focused = focusList
					return m, nil
				}
				if m.isBoardView {
					m.isBoardView = false
					m.focused = focusList
					return m, nil
				}
				if m.isActionableView {
					m.isActionableView = false
					m.focused = focusList
					return m, nil
				}
				if m.isHistoryView {
					m.isHistoryView = false
					m.focused = focusList
					return m, nil
				}
				// At main list - show quit confirmation
				m.showQuitConfirm = true
				m.focused = focusQuitConfirm
				return m, nil

			case "tab":
				if m.isSplitView && !m.isBoardView {
					if m.focused == focusList {
						m.focused = focusDetail
						// Update viewport when switching to detail view
						m.updateViewportContent()
					} else {
						m.focused = focusList
					}
				}

			case "b":
				m.clearAttentionOverlay()
				m.isBoardView = !m.isBoardView
				m.isGraphView = false
				m.isActionableView = false
				if m.isBoardView {
					m.focused = focusBoard
				} else {
					m.focused = focusList
				}

			case "g":
				// Toggle graph view
				m.clearAttentionOverlay()
				m.isGraphView = !m.isGraphView
				m.isBoardView = false
				m.isActionableView = false
				if m.isGraphView {
					m.focused = focusGraph
				} else {
					m.focused = focusList
				}
				return m, nil

			case "a":
				// Toggle actionable view
				m.clearAttentionOverlay()
				m.isActionableView = !m.isActionableView
				m.isGraphView = false
				m.isBoardView = false
				if m.isActionableView {
					// Build execution plan
					analyzer := analysis.NewAnalyzer(m.issues)
					plan := analyzer.GetExecutionPlan()
					m.actionableView = NewActionableModel(plan, m.theme)
					m.actionableView.SetSize(m.width, m.height-2)
					m.focused = focusActionable
				} else {
					m.focused = focusList
				}
				return m, nil

			case "i":
				m.clearAttentionOverlay()
				if m.focused == focusInsights {
					m.focused = focusList
				} else {
					m.focused = focusInsights
					m.isGraphView = false
					m.isBoardView = false
					m.isActionableView = false
					m.focused = focusInsights
					// Refresh insights using latest analysis snapshot
					if m.analysis != nil {
						ins := m.analysis.GenerateInsights(len(m.issues))
						m.insightsPanel = NewInsightsModel(ins, m.issueMap, m.theme)
						// Include priority triage (bv-91)
						triage := analysis.ComputeTriage(m.issues)
						m.insightsPanel.SetTopPicks(triage.QuickRef.TopPicks)
						// Set full recommendations with breakdown for priority radar (bv-93)
						dataHash := fmt.Sprintf("v%s@%s#%d", triage.Meta.Version, triage.Meta.GeneratedAt.Format("15:04:05"), triage.Meta.IssueCount)
						m.insightsPanel.SetRecommendations(triage.Recommendations, dataHash)
						panelHeight := m.height - 2
						if panelHeight < 3 {
							panelHeight = 3
						}
						m.insightsPanel.SetSize(m.width, panelHeight)
					}
				}
				return m, nil

			case "p":
				// Toggle priority hints
				m.showPriorityHints = !m.showPriorityHints
				// Update delegate with new state
				m.list.SetDelegate(IssueDelegate{
					Theme:             m.theme,
					ShowPriorityHints: m.showPriorityHints,
					PriorityHints:     m.priorityHints,
					WorkspaceMode:     m.workspaceMode,
				})
				return m, nil

			case "H":
				// Toggle history view
				m.clearAttentionOverlay()
				m.isHistoryView = !m.isHistoryView
				m.isGraphView = false
				m.isBoardView = false
				m.isActionableView = false
				if m.isHistoryView {
					// Ensure history model has latest sizing
					bodyHeight := m.height - 1
					if bodyHeight < 5 {
						bodyHeight = 5
					}
					m.historyView.SetSize(m.width, bodyHeight)
					m.focused = focusHistory
				} else {
					m.focused = focusList
				}
				return m, nil

			case "L":
				// Open label dashboard (phase 1: table view)
				m.clearAttentionOverlay()
				m.isGraphView = false
				m.isBoardView = false
				m.isActionableView = false
				m.focused = focusLabelDashboard
				// Compute label health (fast; phase1 metrics only needed) with caching
				if !m.labelHealthCached {
					cfg := analysis.DefaultLabelHealthConfig()
					m.labelHealthCache = analysis.ComputeAllLabelHealth(m.issues, cfg, time.Now().UTC(), m.analysis)
					m.labelHealthCached = true
				}
				m.labelDashboard.SetData(m.labelHealthCache.Labels)
				m.labelDashboard.SetSize(m.width, m.height-1)
				m.statusMsg = fmt.Sprintf("Labels: %d total • critical %d • warning %d", m.labelHealthCache.TotalLabels, m.labelHealthCache.CriticalCount, m.labelHealthCache.WarningCount)
				m.statusIsError = false
				return m, nil

			case "A":
				// Attention view: compute attention scores (cached) and render as text
				if !m.attentionCached {
					cfg := analysis.DefaultLabelHealthConfig()
					m.attentionCache = analysis.ComputeLabelAttentionScores(m.issues, cfg, time.Now().UTC())
					m.attentionCached = true
				}
				attText, _ := ComputeAttentionView(m.issues, max(40, m.width-4))
				m.isGraphView = false
				m.isBoardView = false
				m.isActionableView = false
				m.focused = focusInsights
				m.showAttentionView = true
				m.insightsPanel = NewInsightsModel(analysis.Insights{}, m.issueMap, m.theme)
				m.insightsPanel.labelAttention = m.attentionCache.Labels
				m.insightsPanel.extraText = attText
				panelHeight := m.height - 2
				if panelHeight < 3 {
					panelHeight = 3
				}
				m.insightsPanel.SetSize(m.width, panelHeight)
				return m, nil

			case "F":
				// Flow matrix view (cross-label dependencies)
				m.clearAttentionOverlay()
				cfg := analysis.DefaultLabelHealthConfig()
				flow := analysis.ComputeCrossLabelFlow(m.issues, cfg)
				m.flowMatrixText = FlowMatrixView(flow, max(60, m.width-4))
				m.isGraphView = false
				m.isBoardView = false
				m.isActionableView = false
				m.focused = focusInsights
				m.insightsPanel = NewInsightsModel(analysis.Insights{}, m.issueMap, m.theme)
				m.insightsPanel.labelFlow = &flow
				m.insightsPanel.extraText = m.flowMatrixText
				panelHeight := m.height - 2
				if panelHeight < 3 {
					panelHeight = 3
				}
				m.insightsPanel.SetSize(m.width, panelHeight)
				return m, nil

			case "!":
				// Toggle alerts panel (bv-168)
				// Only show if there are active alerts
				activeCount := 0
				for _, a := range m.alerts {
					if !m.dismissedAlerts[alertKey(a)] {
						activeCount++
					}
				}
				if activeCount > 0 {
					m.showAlertsPanel = !m.showAlertsPanel
					m.alertsCursor = 0 // Reset cursor when opening
				} else {
					m.statusMsg = "No active alerts"
					m.statusIsError = false
				}
				return m, nil

			case "R":
				// Toggle recipe picker overlay
				m.showRecipePicker = !m.showRecipePicker
				if m.showRecipePicker {
					m.recipePicker.SetSize(m.width, m.height-1)
					m.focused = focusRecipePicker
				} else {
					m.focused = focusList
				}
				return m, nil

			case "w":
				// Toggle repo picker overlay (workspace mode)
				if !m.workspaceMode || len(m.availableRepos) == 0 {
					m.statusMsg = "Repo filter available only in workspace mode"
					m.statusIsError = false
					return m, nil
				}
				m.showRepoPicker = !m.showRepoPicker
				if m.showRepoPicker {
					m.repoPicker = NewRepoPickerModel(m.availableRepos, m.theme)
					m.repoPicker.SetActiveRepos(m.activeRepos)
					m.repoPicker.SetSize(m.width, m.height-1)
					m.focused = focusRepoPicker
				} else {
					m.focused = focusList
				}
				return m, nil

			case "E":
				// Export to Markdown file
				m.exportToMarkdown()
				return m, nil

			case "l":
				// Open label picker for quick filter (bv-126)
				if len(m.issues) == 0 {
					return m, nil
				}
				// Update labels in case they changed
				labelExtraction := analysis.ExtractLabels(m.issues)
				m.labelPicker.SetLabels(labelExtraction.Labels)
				m.labelPicker.Reset()
				m.labelPicker.SetSize(m.width, m.height-1)
				m.showLabelPicker = true
				m.focused = focusLabelPicker
				return m, nil

			}

			// Focus-specific key handling
			switch m.focused {
			case focusRecipePicker:
				m = m.handleRecipePickerKeys(msg)

			case focusRepoPicker:
				m = m.handleRepoPickerKeys(msg)

			case focusLabelPicker:
				m = m.handleLabelPickerKeys(msg)

			case focusInsights:
				m = m.handleInsightsKeys(msg)

			case focusBoard:
				m = m.handleBoardKeys(msg)

			case focusLabelDashboard:
				if selectedLabel, cmd := m.labelDashboard.Update(msg); selectedLabel != "" {
					// Filter list by selected label and jump back to list view
					m.currentFilter = "label:" + selectedLabel
					m.applyFilter()
					m.focused = focusList
					return m, cmd
				}
				// Open detail modal on 'h'
				if msg.String() == "h" && len(m.labelDashboard.labels) > 0 {
					idx := m.labelDashboard.cursor
					if idx >= 0 && idx < len(m.labelDashboard.labels) {
						lh := m.labelDashboard.labels[idx]
						m.showLabelHealthDetail = true
						m.labelHealthDetail = &lh
						// Precompute cross-label flows for this label
						m.labelHealthDetailFlow = m.getCrossFlowsForLabel(lh.Label)
						return m, nil
					}
				}
				// Open drilldown overlay on 'd'
				if msg.String() == "d" && len(m.labelDashboard.labels) > 0 {
					idx := m.labelDashboard.cursor
					if idx >= 0 && idx < len(m.labelDashboard.labels) {
						lh := m.labelDashboard.labels[idx]
						m.labelDrilldownLabel = lh.Label
						m.labelDrilldownIssues = m.filterIssuesByLabel(lh.Label)
						m.showLabelDrilldown = true
						return m, nil
					}
				}

			case focusGraph:
				m = m.handleGraphKeys(msg)

			case focusActionable:
				m = m.handleActionableKeys(msg)

			case focusHistory:
				m = m.handleHistoryKeys(msg)

			case focusSprint:
				m = m.handleSprintKeys(msg)

			case focusList:
				m = m.handleListKeys(msg)

			case focusDetail:
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.MouseMsg:
		// Handle mouse wheel scrolling
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			// Scroll up based on current focus
			switch m.focused {
			case focusList:
				if m.list.Index() > 0 {
					m.list.Select(m.list.Index() - 1)
					// Don't auto-update viewport - prevents lag
				}
			case focusDetail:
				m.viewport.ScrollUp(3)
			case focusInsights:
				m.insightsPanel.MoveUp()
			case focusBoard:
				m.board.MoveUp()
			case focusGraph:
				m.graphView.PageUp()
			case focusActionable:
				m.actionableView.MoveUp()
			case focusHistory:
				m.historyView.MoveUp()
			}
			return m, nil
		case tea.MouseButtonWheelDown:
			// Scroll down based on current focus
			switch m.focused {
			case focusList:
				if m.list.Index() < len(m.list.Items())-1 {
					m.list.Select(m.list.Index() + 1)
					// Don't auto-update viewport - prevents lag
				}
			case focusDetail:
				m.viewport.ScrollDown(3)
			case focusInsights:
				m.insightsPanel.MoveDown()
			case focusBoard:
				m.board.MoveDown()
			case focusGraph:
				m.graphView.PageDown()
			case focusActionable:
				m.actionableView.MoveDown()
			case focusHistory:
				m.historyView.MoveDown()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.isSplitView = msg.Width > SplitViewThreshold
		m.ready = true
		bodyHeight := m.height - 1 // keep 1 row for footer
		if bodyHeight < 5 {
			bodyHeight = 5
		}

		if m.isSplitView {
			// Calculate dimensions accounting for 2 panels with borders(2)+padding(2) = 4 overhead each
			// Total overhead = 8
			availWidth := msg.Width - 8
			if availWidth < 10 {
				availWidth = 10
			}

			listInnerWidth := int(float64(availWidth) * 0.4)
			detailInnerWidth := availWidth - listInnerWidth

			// listHeight fits header (1) + page line (1) inside a panel with Border (2)
			listHeight := bodyHeight - 4
			if listHeight < 3 {
				listHeight = 3
			}

			m.list.SetSize(listInnerWidth, listHeight)
			m.viewport = viewport.New(detailInnerWidth, bodyHeight-2) // Account for border

			m.renderer.SetWidthWithTheme(detailInnerWidth, m.theme)
		} else {
			listHeight := bodyHeight - 2
			if listHeight < 3 {
				listHeight = 3
			}
			m.list.SetSize(msg.Width, listHeight)
			m.viewport = viewport.New(msg.Width, bodyHeight-1)

			// Update renderer for full width
			m.renderer.SetWidthWithTheme(msg.Width, m.theme)
		}

		m.list.SetDelegate(IssueDelegate{
			Theme:             m.theme,
			ShowPriorityHints: m.showPriorityHints,
			PriorityHints:     m.priorityHints,
			WorkspaceMode:     m.workspaceMode,
		})

		// Resize label dashboard table and modal overlay sizing
		m.labelDashboard.SetSize(m.width, bodyHeight)

		m.insightsPanel.SetSize(m.width, bodyHeight)
		m.updateViewportContent()
	}

	// Update list for navigation, but NOT for WindowSizeMsg
	// (we handle sizing ourselves to account for header/footer)
	// Only forward keyboard messages to list when list has focus (bv-hmkz fix)
	// This prevents j/k keys in detail view from changing list selection
	if m.focused == focusList {
		if _, isWindowSize := msg.(tea.WindowSizeMsg); !isWindowSize {
			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Don't auto-update viewport on every keypress - only update when user switches focus with Tab
	// This prevents lag with large task lists during rapid navigation

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var body string

	// Quit confirmation overlay takes highest priority
	if m.showQuitConfirm {
		body = m.renderQuitConfirm()
	} else if m.showLabelHealthDetail && m.labelHealthDetail != nil {
		body = m.renderLabelHealthDetail(*m.labelHealthDetail)
	} else if m.showLabelGraphAnalysis && m.labelGraphAnalysisResult != nil {
		body = m.renderLabelGraphAnalysis()
	} else if m.showLabelDrilldown && m.labelDrilldownLabel != "" {
		body = m.renderLabelDrilldown()
	} else if m.showAlertsPanel {
		body = m.renderAlertsPanel()
	} else if m.showTimeTravelPrompt {
		body = m.renderTimeTravelPrompt()
	} else if m.showRecipePicker {
		body = m.recipePicker.View()
	} else if m.showRepoPicker {
		body = m.repoPicker.View()
	} else if m.showLabelPicker {
		body = m.labelPicker.View()
	} else if m.showHelp {
		body = m.renderHelpOverlay()
	} else if m.focused == focusInsights {
		body = m.insightsPanel.View()
	} else if m.isGraphView {
		body = m.graphView.View(m.width, m.height-1)
	} else if m.isBoardView {
		body = m.board.View(m.width, m.height-1)
	} else if m.isActionableView {
		m.actionableView.SetSize(m.width, m.height-2)
		body = m.actionableView.Render()
	} else if m.isHistoryView {
		m.historyView.SetSize(m.width, m.height-1)
		body = m.historyView.View()
	} else if m.isSprintView {
		body = m.sprintViewText
	} else if m.isSplitView {
		body = m.renderSplitView()
	} else if m.focused == focusLabelDashboard {
		m.labelDashboard.SetSize(m.width, m.height-1)
		body = m.labelDashboard.View()
	} else {
		// Mobile view
		if m.showDetails {
			body = m.viewport.View()
		} else {
			body = m.renderListWithHeader()
		}
	}

	// Add shortcuts sidebar if enabled (bv-3qi5)
	if m.showShortcutsSidebar {
		// Update sidebar context based on current focus
		m.shortcutsSidebar.SetContext(ContextFromFocus(m.focused))
		m.shortcutsSidebar.SetSize(m.shortcutsSidebar.Width(), m.height-2)
		sidebar := m.shortcutsSidebar.View()
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, sidebar)
	}

	footer := m.renderFooter()

	// Ensure the final output fits exactly in the terminal height
	// This prevents the header from being pushed off the top
	finalStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		MaxHeight(m.height)

	return finalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, body, footer))
}

// GetTypeIconMD returns the emoji icon for an issue type (for markdown)
func GetTypeIconMD(t string) string {
	switch t {
	case "bug":
		return "🐛"
	case "feature":
		return "✨"
	case "task":
		return "📋"
	case "epic":
		return "🚀" // Use rocket instead of mountain - VS-16 variation selector causes width issues
	case "chore":
		return "🧹"
	default:
		return "•"
	}
}

// Stop cleans up resources (file watcher, etc.)
// Should be called when the program exits
func (m *Model) Stop() {
	if m.watcher != nil {
		m.watcher.Stop()
	}
}

