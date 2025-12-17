package ui

import (
	"fmt"
	"os"

	"github.com/Dicklesworthstone/beads_viewer/pkg/analysis"
	"github.com/Dicklesworthstone/beads_viewer/pkg/loader"
	"github.com/charmbracelet/lipgloss"
)

// enterTimeTravelMode loads historical data and computes diff
func (m *Model) enterTimeTravelMode(revision string) {
	cwd, err := os.Getwd()
	if err != nil {
		m.statusMsg = "❌ Time-travel failed: cannot get working directory"
		m.statusIsError = true
		return
	}

	gitLoader := loader.NewGitLoader(cwd)

	// Check if we're in a git repo first
	if _, err := gitLoader.ResolveRevision("HEAD"); err != nil {
		m.statusMsg = "❌ Time-travel requires a git repository"
		m.statusIsError = true
		return
	}

	// Check if beads files exist at the revision
	hasBeads, err := gitLoader.HasBeadsAtRevision(revision)
	if err != nil || !hasBeads {
		m.statusMsg = fmt.Sprintf("❌ No beads history at %s (try fewer commits back)", revision)
		m.statusIsError = true
		return
	}

	// Load historical issues
	historicalIssues, err := gitLoader.LoadAt(revision)
	if err != nil {
		m.statusMsg = fmt.Sprintf("❌ Time-travel failed: %v", err)
		m.statusIsError = true
		return
	}

	// Create snapshots and compute diff
	fromSnapshot := analysis.NewSnapshot(historicalIssues)
	toSnapshot := analysis.NewSnapshot(m.issues)
	diff := analysis.CompareSnapshots(fromSnapshot, toSnapshot)

	// Build lookup sets for badges
	m.newIssueIDs = make(map[string]bool)
	for _, issue := range diff.NewIssues {
		m.newIssueIDs[issue.ID] = true
	}

	m.closedIssueIDs = make(map[string]bool)
	for _, issue := range diff.ClosedIssues {
		m.closedIssueIDs[issue.ID] = true
	}

	m.modifiedIssueIDs = make(map[string]bool)
	for _, mod := range diff.ModifiedIssues {
		m.modifiedIssueIDs[mod.IssueID] = true
	}

	m.timeTravelMode = true
	m.timeTravelDiff = diff
	m.timeTravelSince = revision

	// Success feedback
	m.statusMsg = fmt.Sprintf("⏱️ Time-travel: comparing with %s (+%d ✅%d ~%d)",
		revision, diff.Summary.IssuesAdded, diff.Summary.IssuesClosed, diff.Summary.IssuesModified)
	m.statusIsError = false

	// Rebuild list items with diff info
	m.rebuildListWithDiffInfo()
}

// exitTimeTravelMode clears time-travel state
func (m *Model) exitTimeTravelMode() {
	m.timeTravelMode = false
	m.timeTravelDiff = nil
	m.timeTravelSince = ""
	m.newIssueIDs = nil
	m.closedIssueIDs = nil
	m.modifiedIssueIDs = nil

	// Feedback
	m.statusMsg = "⏱️ Time-travel mode disabled"
	m.statusIsError = false

	// Rebuild list without diff info
	m.rebuildListWithDiffInfo()
}

// rebuildListWithDiffInfo recreates list items with current diff state
func (m *Model) rebuildListWithDiffInfo() {
	if m.activeRecipe != nil {
		m.applyRecipe(m.activeRecipe)
	} else {
		m.applyFilter()
	}
}

// IsTimeTravelMode returns whether time-travel mode is active
func (m Model) IsTimeTravelMode() bool {
	return m.timeTravelMode
}

// TimeTravelDiff returns the current diff (nil if not in time-travel mode)
func (m Model) TimeTravelDiff() *analysis.SnapshotDiff {
	return m.timeTravelDiff
}

// getDiffStatus returns the diff status for an issue if time-travel mode is active
func (m Model) getDiffStatus(id string) DiffStatus {
	if !m.timeTravelMode {
		return DiffStatusNone
	}
	if m.newIssueIDs[id] {
		return DiffStatusNew
	}
	if m.closedIssueIDs[id] {
		return DiffStatusClosed
	}
	if m.modifiedIssueIDs[id] {
		return DiffStatusModified
	}
	return DiffStatusNone
}

// renderTimeTravelPrompt renders the time-travel revision input overlay
func (m Model) renderTimeTravelPrompt() string {
	t := m.theme

	boxStyle := t.Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 3).
		Align(lipgloss.Center)

	titleStyle := t.Renderer.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	subtitleStyle := t.Renderer.NewStyle().
		Foreground(t.Subtext).
		Italic(true)

	exampleStyle := t.Renderer.NewStyle().
		Foreground(t.Secondary)

	keyStyle := t.Renderer.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	textStyle := t.Renderer.NewStyle().
		Foreground(t.Base.GetForeground())

	// Build content
	content := titleStyle.Render("⏱️  Time-Travel Mode") + "\n\n" +
		subtitleStyle.Render("Compare current state with a historical revision") + "\n\n" +
		m.timeTravelInput.View() + "\n\n" +
		exampleStyle.Render("Examples: HEAD~5, main, v1.0.0, 2024-01-01, abc123") + "\n\n" +
		textStyle.Render("Press ") + keyStyle.Render("Enter") + textStyle.Render(" to compare, ") +
		keyStyle.Render("Esc") + textStyle.Render(" to cancel")

	box := boxStyle.Render(content)

	return lipgloss.Place(
		m.width,
		m.height-1,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}
