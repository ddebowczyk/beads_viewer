package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/Dicklesworthstone/beads_viewer/pkg/correlation"
)

// getEventIcon returns the icon for a correlation event type
func getEventIcon(eventType correlation.EventType) string {
	switch eventType {
	case correlation.EventCreated:
		return "🟢"
	case correlation.EventClaimed:
		return "🔵"
	case correlation.EventClosed:
		return "⚫"
	case correlation.EventReopened:
		return "🟡"
	case correlation.EventModified:
		return "📝"
	default:
		return "•"
	}
}

// truncateString truncates a string to maxLen runes with ellipsis.
// Uses rune-based counting to safely handle UTF-8 multi-byte characters.
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
}

// renderBeadHistoryMD renders the history section for a bead's detail view
func (m *Model) renderBeadHistoryMD(beadID string) string {
	hist := m.historyView.GetHistoryForBead(beadID)
	if hist == nil || len(hist.Commits) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("### 📜 History\n\n")

	// Lifecycle milestones from events
	if len(hist.Events) > 0 {
		sb.WriteString("**Lifecycle:**\n")
		for _, event := range hist.Events {
			icon := getEventIcon(event.EventType)
			sb.WriteString(fmt.Sprintf("- %s **%s** %s by %s\n",
				icon,
				event.EventType,
				event.Timestamp.Format("Jan 02 15:04"),
				event.Author,
			))
		}
		sb.WriteString("\n")
	}

	// Correlated commits
	sb.WriteString(fmt.Sprintf("**Related Commits (%d):**\n", len(hist.Commits)))
	for i, commit := range hist.Commits {
		if i >= 5 {
			sb.WriteString(fmt.Sprintf("  ... and %d more commits\n", len(hist.Commits)-5))
			break
		}

		// Confidence indicator
		confIcon := "🟢"
		if commit.Confidence < 0.5 {
			confIcon = "🟡"
		} else if commit.Confidence < 0.8 {
			confIcon = "🟠"
		}

		sb.WriteString(fmt.Sprintf("- %s **%.0f%%** `%s` %s\n",
			confIcon,
			commit.Confidence*100,
			commit.ShortSHA,
			truncateString(commit.Message, 40),
		))

		// Show files for high-confidence commits
		if commit.Confidence >= 0.8 && len(commit.Files) > 0 && len(commit.Files) <= 3 {
			for _, f := range commit.Files {
				sb.WriteString(fmt.Sprintf("  - `%s` (+%d, -%d)\n", f.Path, f.Insertions, f.Deletions))
			}
		}
	}

	sb.WriteString("\n*Press H for full history view*\n\n")
	return sb.String()
}

// enterHistoryView loads correlation data and shows the history view
func (m *Model) enterHistoryView() {
	cwd, err := os.Getwd()
	if err != nil {
		m.statusMsg = "Cannot get working directory for history"
		m.statusIsError = true
		return
	}

	// Convert model.Issue to correlation.BeadInfo
	beads := make([]correlation.BeadInfo, len(m.issues))
	for i, issue := range m.issues {
		beads[i] = correlation.BeadInfo{
			ID:     issue.ID,
			Title:  issue.Title,
			Status: string(issue.Status),
		}
	}

	// Load correlation data
	correlator := correlation.NewCorrelator(cwd, m.beadsPath)
	opts := correlation.CorrelatorOptions{
		Limit: 500, // Reasonable limit for TUI performance
	}

	report, err := correlator.GenerateReport(beads, opts)
	if err != nil {
		m.statusMsg = fmt.Sprintf("History load failed: %v", err)
		m.statusIsError = true
		return
	}

	// Initialize or update history view
	m.historyView = NewHistoryModel(report, m.theme)
	m.historyView.SetSize(m.width, m.height-1)
	m.isHistoryView = true
	m.focused = focusHistory

	m.statusMsg = fmt.Sprintf("Loaded history: %d beads with commits", report.Stats.BeadsWithCommits)
	m.statusIsError = false
}
