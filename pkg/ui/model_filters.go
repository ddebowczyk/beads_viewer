package ui

import (
	"sort"
	"strings"

	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
	"github.com/Dicklesworthstone/beads_viewer/pkg/recipe"
	"github.com/charmbracelet/bubbles/list"
)

func (m *Model) updateSemanticIDs(items []list.Item) {
	if m.semanticSearch == nil {
		return
	}
	ids := make([]string, 0, len(items))
	for _, it := range items {
		if issueItem, ok := it.(IssueItem); ok {
			ids = append(ids, issueItem.Issue.ID)
		}
	}
	m.semanticSearch.SetIDs(ids)
}

func (m *Model) applyFilter() {
	var filteredItems []list.Item
	var filteredIssues []model.Issue

	for _, issue := range m.issues {
		// Workspace repo filter (nil = all repos)
		if m.workspaceMode && m.activeRepos != nil {
			repoKey := strings.ToLower(ExtractRepoPrefix(issue.ID))
			if repoKey != "" && !m.activeRepos[repoKey] {
				continue
			}
		}

		include := false
		switch m.currentFilter {
		case "all":
			include = true
		case "open":
			include = issue.Status != model.StatusClosed
		case "closed":
			include = issue.Status == model.StatusClosed
		case "ready":
			// Ready = Open/InProgress AND NO Open Blockers
			if issue.Status != model.StatusClosed && issue.Status != model.StatusBlocked {
				isBlocked := false
				for _, dep := range issue.Dependencies {
					if dep.Type == model.DepBlocks {
						if blocker, exists := m.issueMap[dep.DependsOnID]; exists && blocker.Status != model.StatusClosed {
							isBlocked = true
							break
						}
					}
				}
				include = !isBlocked
			}
		default:
			if strings.HasPrefix(m.currentFilter, "label:") {
				label := strings.TrimPrefix(m.currentFilter, "label:")
				for _, l := range issue.Labels {
					if l == label {
						include = true
						break
					}
				}
			}
		}

		if include {
			// Use pre-computed graph scores (avoid redundant calculation)
			item := IssueItem{
				Issue:      issue,
				GraphScore: m.analysis.GetPageRankScore(issue.ID),
				Impact:     m.analysis.GetCriticalPathScore(issue.ID),
				DiffStatus: m.getDiffStatus(issue.ID),
				RepoPrefix: ExtractRepoPrefix(issue.ID),
			}
			// Add triage data (bv-151)
			item.TriageScore = m.triageScores[issue.ID]
			if reasons, exists := m.triageReasons[issue.ID]; exists {
				item.TriageReason = reasons.Primary
				item.TriageReasons = reasons.All
			}
			item.IsQuickWin = m.quickWinSet[issue.ID]
			item.IsBlocker = m.blockerSet[issue.ID]
			item.UnblocksCount = len(m.unblocksMap[issue.ID])
			filteredItems = append(filteredItems, item)
			filteredIssues = append(filteredIssues, issue)
		}
	}

	m.list.SetItems(filteredItems)
	m.updateSemanticIDs(filteredItems)
	m.board.SetIssues(filteredIssues)
	// Generate insights for graph view (for metric rankings and sorting)
	filterIns := m.analysis.GenerateInsights(len(filteredIssues))
	m.graphView.SetIssues(filteredIssues, &filterIns)

	// Keep selection in bounds
	if len(filteredItems) > 0 && m.list.Index() >= len(filteredItems) {
		m.list.Select(0)
	}
	m.updateViewportContent()
}

// applyRecipe applies a recipe's filters and sort to the current view
func (m *Model) applyRecipe(r *recipe.Recipe) {
	if r == nil {
		return
	}

	var filteredItems []list.Item
	var filteredIssues []model.Issue

	for _, issue := range m.issues {
		include := true

		// Workspace repo filter (nil = all repos)
		if m.workspaceMode && m.activeRepos != nil {
			repoKey := strings.ToLower(ExtractRepoPrefix(issue.ID))
			if repoKey != "" && !m.activeRepos[repoKey] {
				include = false
			}
		}

		// Apply status filter
		if len(r.Filters.Status) > 0 {
			statusMatch := false
			for _, s := range r.Filters.Status {
				if string(issue.Status) == s {
					statusMatch = true
					break
				}
			}
			include = include && statusMatch
		}

		// Apply priority filter
		if include && len(r.Filters.Priority) > 0 {
			prioMatch := false
			for _, p := range r.Filters.Priority {
				if issue.Priority == p {
					prioMatch = true
					break
				}
			}
			include = include && prioMatch
		}

		// Apply tags filter (must have ALL specified tags)
		if include && len(r.Filters.Tags) > 0 {
			labelSet := make(map[string]bool)
			for _, l := range issue.Labels {
				labelSet[l] = true
			}
			for _, required := range r.Filters.Tags {
				if !labelSet[required] {
					include = false
					break
				}
			}
		}

		// Apply actionable filter
		if include && r.Filters.Actionable != nil && *r.Filters.Actionable {
			// Check if issue is blocked
			isBlocked := false
			for _, dep := range issue.Dependencies {
				if dep.Type == model.DepBlocks {
					if blocker, exists := m.issueMap[dep.DependsOnID]; exists && blocker.Status != model.StatusClosed {
						isBlocked = true
						break
					}
				}
			}
			include = !isBlocked
		}

		if include {
			item := IssueItem{
				Issue:      issue,
				GraphScore: m.analysis.GetPageRankScore(issue.ID),
				Impact:     m.analysis.GetCriticalPathScore(issue.ID),
				DiffStatus: m.getDiffStatus(issue.ID),
				RepoPrefix: ExtractRepoPrefix(issue.ID),
			}
			// Add triage data (bv-151)
			item.TriageScore = m.triageScores[issue.ID]
			if reasons, exists := m.triageReasons[issue.ID]; exists {
				item.TriageReason = reasons.Primary
				item.TriageReasons = reasons.All
			}
			item.IsQuickWin = m.quickWinSet[issue.ID]
			item.IsBlocker = m.blockerSet[issue.ID]
			item.UnblocksCount = len(m.unblocksMap[issue.ID])
			filteredItems = append(filteredItems, item)
			filteredIssues = append(filteredIssues, issue)
		}
	}

	// Apply sort
	descending := r.Sort.Direction == "desc"
	if r.Sort.Field != "" {
		sort.Slice(filteredItems, func(i, j int) bool {
			iItem := filteredItems[i].(IssueItem)
			jItem := filteredItems[j].(IssueItem)
			less := false

			switch r.Sort.Field {
			case "priority":
				less = iItem.Issue.Priority < jItem.Issue.Priority
			case "created", "created_at":
				less = iItem.Issue.CreatedAt.Before(jItem.Issue.CreatedAt)
			case "updated", "updated_at":
				less = iItem.Issue.UpdatedAt.Before(jItem.Issue.UpdatedAt)
			case "impact":
				// Use analysis map for sort
				less = m.analysis.GetCriticalPathScore(iItem.Issue.ID) < m.analysis.GetCriticalPathScore(jItem.Issue.ID)
			case "pagerank":
				// Use analysis map for sort
				less = m.analysis.GetPageRankScore(iItem.Issue.ID) < m.analysis.GetPageRankScore(jItem.Issue.ID)
			default:
				less = iItem.Issue.Priority < jItem.Issue.Priority
			}

			if descending {
				return !less
			}
			return less
		})

		// Re-sort issues list too
		sort.Slice(filteredIssues, func(i, j int) bool {
			less := false
			switch r.Sort.Field {
			case "priority":
				less = filteredIssues[i].Priority < filteredIssues[j].Priority
			case "created", "created_at":
				less = filteredIssues[i].CreatedAt.Before(filteredIssues[j].CreatedAt)
			case "updated", "updated_at":
				less = filteredIssues[i].UpdatedAt.Before(filteredIssues[j].UpdatedAt)
			case "impact":
				// Use analysis map for sort
				less = m.analysis.GetCriticalPathScore(filteredIssues[i].ID) < m.analysis.GetCriticalPathScore(filteredIssues[j].ID)
			case "pagerank":
				// Use analysis map for sort
				less = m.analysis.GetPageRankScore(filteredIssues[i].ID) < m.analysis.GetPageRankScore(filteredIssues[j].ID)
			default:
				less = filteredIssues[i].Priority < filteredIssues[j].Priority
			}
			if descending {
				return !less
			}
			return less
		})
	}

	m.list.SetItems(filteredItems)
	m.updateSemanticIDs(filteredItems)
	m.board.SetIssues(filteredIssues)
	// Generate insights for graph view (for metric rankings and sorting)
	recipeIns := m.analysis.GenerateInsights(len(filteredIssues))
	m.graphView.SetIssues(filteredIssues, &recipeIns)

	// Update filter indicator
	m.currentFilter = "recipe:" + r.Name

	// Keep selection in bounds
	if len(filteredItems) > 0 && m.list.Index() >= len(filteredItems) {
		m.list.Select(0)
	}
	m.updateViewportContent()
}

// SetFilter sets the current filter and applies it (exposed for testing)
func (m *Model) SetFilter(f string) {
	m.currentFilter = f
	m.applyFilter()
}

// FilteredIssues returns the currently visible issues (exposed for testing)
func (m Model) FilteredIssues() []model.Issue {
	items := m.list.Items()
	issues := make([]model.Issue, 0, len(items))
	for _, item := range items {
		if issueItem, ok := item.(IssueItem); ok {
			issues = append(issues, issueItem.Issue)
		}
	}
	return issues
}
