package ui

import (
	"fmt"
	"strings"
)

func (m *Model) updateViewportContent() {
	selectedItem := m.list.SelectedItem()
	if selectedItem == nil {
		m.viewport.SetContent("No issues selected")
		return
	}

	// Safe type assertion
	issueItem, ok := selectedItem.(IssueItem)
	if !ok {
		m.viewport.SetContent("Error: invalid item type")
		return
	}
	item := issueItem.Issue

	var sb strings.Builder

	if m.updateAvailable {
		sb.WriteString(fmt.Sprintf("⭐ **Update Available:** [%s](%s)\n\n", m.updateTag, m.updateURL))
	}

	// Title Block
	sb.WriteString(fmt.Sprintf("# %s %s\n", GetTypeIconMD(string(item.IssueType)), item.Title))

	// Meta Table
	sb.WriteString("| ID | Status | Priority | Assignee | Created |\n|---|---|---|---|---|\n")
	sb.WriteString(fmt.Sprintf("| **%s** | **%s** | %s | @%s | %s |\n\n",
		item.ID,
		strings.ToUpper(string(item.Status)),
		GetPriorityIcon(item.Priority),
		item.Assignee,
		item.CreatedAt.Format("2006-01-02"),
	))

	// Labels (bv-f103 fix: display labels in detail view)
	if len(item.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("**Labels:** %s\n\n", strings.Join(item.Labels, ", ")))
	}

	// Triage Insights (bv-151)
	if issueItem.TriageScore > 0 || issueItem.TriageReason != "" || issueItem.UnblocksCount > 0 || issueItem.IsQuickWin || issueItem.IsBlocker {
		sb.WriteString("### 🎯 Triage Insights\n")

		// Score with visual indicator
		scoreIcon := "🔵"
		if issueItem.TriageScore >= 0.7 {
			scoreIcon = "🔴"
		} else if issueItem.TriageScore >= 0.4 {
			scoreIcon = "🟠"
		}
		sb.WriteString(fmt.Sprintf("- **Triage Score:** %s %.2f/1.00\n", scoreIcon, issueItem.TriageScore))

		// Special flags
		if issueItem.IsQuickWin {
			sb.WriteString("- **⭐ Quick Win** — Low effort, high impact opportunity\n")
		}
		if issueItem.IsBlocker {
			sb.WriteString("- **🔴 Critical Blocker** — Completing this unblocks significant downstream work\n")
		}

		// Unblocks count
		if issueItem.UnblocksCount > 0 {
			sb.WriteString(fmt.Sprintf("- **🔓 Unblocks:** %d downstream items when completed\n", issueItem.UnblocksCount))
		}

		// Primary reason
		if issueItem.TriageReason != "" {
			sb.WriteString(fmt.Sprintf("- **Primary Reason:** %s\n", issueItem.TriageReason))
		}

		// All reasons (if multiple)
		if len(issueItem.TriageReasons) > 1 {
			sb.WriteString("- **All Reasons:**\n")
			for _, reason := range issueItem.TriageReasons {
				sb.WriteString(fmt.Sprintf("  - %s\n", reason))
			}
		}

		sb.WriteString("\n")
	}

	// Graph Analysis (using thread-safe accessors)
	pr := m.analysis.GetPageRankScore(item.ID)
	bt := m.analysis.GetBetweennessScore(item.ID)
	imp := m.analysis.GetCriticalPathScore(item.ID)
	ev := m.analysis.GetEigenvectorScore(item.ID)
	hub := m.analysis.GetHubScore(item.ID)
	auth := m.analysis.GetAuthorityScore(item.ID)

	sb.WriteString("### Graph Analysis\n")
	sb.WriteString(fmt.Sprintf("- **Impact Depth**: %.0f (downstream chain length)\n", imp))
	sb.WriteString(fmt.Sprintf("- **Centrality**: PR %.4f • BW %.4f • EV %.4f\n", pr, bt, ev))
	sb.WriteString(fmt.Sprintf("- **Flow Role**: Hub %.4f • Authority %.4f\n\n", hub, auth))

	// Description
	if item.Description != "" {
		sb.WriteString("### Description\n")
		sb.WriteString(item.Description + "\n\n")
	}

	// Acceptance Criteria
	if item.AcceptanceCriteria != "" {
		sb.WriteString("### Acceptance Criteria\n")
		sb.WriteString(item.AcceptanceCriteria + "\n\n")
	}

	// Notes
	if item.Notes != "" {
		sb.WriteString("### Notes\n")
		sb.WriteString(item.Notes + "\n\n")
	}

	// Dependency Graph (Tree)
	if len(item.Dependencies) > 0 {
		rootNode := BuildDependencyTree(item.ID, m.issueMap, 3) // Max depth 3
		treeStr := RenderDependencyTree(rootNode)
		sb.WriteString("```\n" + treeStr + "```\n\n")
	}

	// Comments
	if len(item.Comments) > 0 {
		sb.WriteString(fmt.Sprintf("### Comments (%d)\n", len(item.Comments)))
		for _, comment := range item.Comments {
			sb.WriteString(fmt.Sprintf("> **%s** (%s)\n> \n> %s\n\n",
				comment.Author,
				FormatTimeRel(comment.CreatedAt),
				strings.ReplaceAll(comment.Text, "\n", "\n> ")))
		}
	}

	// History Section (if data is loaded)
	if m.historyView.HasReport() {
		historyMD := m.renderBeadHistoryMD(item.ID)
		if historyMD != "" {
			sb.WriteString(historyMD)
		}
	}

	rendered, err := m.renderer.Render(sb.String())
	if err != nil {
		m.viewport.SetContent(fmt.Sprintf("Error rendering markdown: %v", err))
	} else {
		m.viewport.SetContent(rendered)
	}
}
