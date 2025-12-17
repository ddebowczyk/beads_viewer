package ui

import (
	"fmt"
)

// WorkspaceInfo contains workspace loading metadata for TUI display
type WorkspaceInfo struct {
	Enabled      bool
	RepoCount    int
	FailedCount  int
	TotalIssues  int
	RepoPrefixes []string
}

// EnableWorkspaceMode configures the model for workspace (multi-repo) view
func (m *Model) EnableWorkspaceMode(info WorkspaceInfo) {
	m.workspaceMode = info.Enabled
	m.availableRepos = normalizeRepoPrefixes(info.RepoPrefixes)
	m.activeRepos = nil // nil means all repos are active

	if info.RepoCount > 0 {
		if info.FailedCount > 0 {
			m.workspaceSummary = fmt.Sprintf("%d/%d repos", info.RepoCount-info.FailedCount, info.RepoCount)
		} else {
			m.workspaceSummary = fmt.Sprintf("%d repos", info.RepoCount)
		}
	}

	// Update delegate to show repo badges
	m.list.SetDelegate(IssueDelegate{
		Theme:             m.theme,
		ShowPriorityHints: m.showPriorityHints,
		PriorityHints:     m.priorityHints,
		WorkspaceMode:     m.workspaceMode,
	})
}

// IsWorkspaceMode returns whether workspace mode is active
func (m Model) IsWorkspaceMode() bool {
	return m.workspaceMode
}
