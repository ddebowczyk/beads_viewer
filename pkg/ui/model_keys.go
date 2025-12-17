package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// handleBoardKeys handles keyboard input when the board view is focused
func (m Model) handleBoardKeys(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "h", "left":
		m.board.MoveLeft()
	case "l", "right":
		m.board.MoveRight()
	case "j", "down":
		m.board.MoveDown()
	case "k", "up":
		m.board.MoveUp()
	case "home":
		m.board.MoveToTop()
	case "G", "end":
		m.board.MoveToBottom()
	case "ctrl+d":
		m.board.PageDown(m.height / 3)
	case "ctrl+u":
		m.board.PageUp(m.height / 3)
	case "enter":
		if selected := m.board.SelectedIssue(); selected != nil {
			// Find and select in list
			for i, item := range m.list.Items() {
				if issueItem, ok := item.(IssueItem); ok && issueItem.Issue.ID == selected.ID {
					m.list.Select(i)
					break
				}
			}
			m.isBoardView = false
			m.focused = focusList
			if m.isSplitView {
				m.focused = focusDetail
			} else {
				m.showDetails = true
				m.focused = focusDetail
			}
			m.updateViewportContent()
		}
	}
	return m
}

// handleGraphKeys handles keyboard input when the graph view is focused
func (m Model) handleGraphKeys(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "h", "left":
		m.graphView.MoveLeft()
	case "l", "right":
		m.graphView.MoveRight()
	case "j", "down":
		m.graphView.MoveDown()
	case "k", "up":
		m.graphView.MoveUp()
	case "ctrl+d", "pgdown":
		m.graphView.PageDown()
	case "ctrl+u", "pgup":
		m.graphView.PageUp()
	case "H":
		m.graphView.ScrollLeft()
	case "L":
		m.graphView.ScrollRight()
	case "enter":
		if selected := m.graphView.SelectedIssue(); selected != nil {
			// Find and select in list
			for i, item := range m.list.Items() {
				if issueItem, ok := item.(IssueItem); ok && issueItem.Issue.ID == selected.ID {
					m.list.Select(i)
					break
				}
			}
			m.isGraphView = false
			m.focused = focusList
			if m.isSplitView {
				m.focused = focusDetail
			} else {
				m.showDetails = true
				m.focused = focusDetail
			}
			m.updateViewportContent()
		}
	}
	return m
}

// handleActionableKeys handles keyboard input when actionable view is focused
func (m Model) handleActionableKeys(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "j", "down":
		m.actionableView.MoveDown()
	case "k", "up":
		m.actionableView.MoveUp()
	case "enter":
		// Jump to selected issue in list view
		selectedID := m.actionableView.SelectedIssueID()
		if selectedID != "" {
			for i, item := range m.list.Items() {
				if issueItem, ok := item.(IssueItem); ok && issueItem.Issue.ID == selectedID {
					m.list.Select(i)
					break
				}
			}
			m.isActionableView = false
			m.focused = focusList
			if m.isSplitView {
				m.focused = focusDetail
			} else {
				m.showDetails = true
				m.focused = focusDetail
			}
			m.updateViewportContent()
		}
	}
	return m
}

// handleHistoryKeys handles keyboard input when history view is focused
func (m Model) handleHistoryKeys(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "j", "down":
		m.historyView.MoveDown()
	case "k", "up":
		m.historyView.MoveUp()
	case "J":
		// Navigate to next commit within bead
		m.historyView.NextCommit()
	case "K":
		// Navigate to previous commit within bead
		m.historyView.PrevCommit()
	case "tab":
		m.historyView.ToggleFocus()
	case "enter":
		// Jump to selected bead in main list
		selectedID := m.historyView.SelectedBeadID()
		if selectedID != "" {
			for i, item := range m.list.Items() {
				if issueItem, ok := item.(IssueItem); ok && issueItem.Issue.ID == selectedID {
					m.list.Select(i)
					break
				}
			}
			m.isHistoryView = false
			m.focused = focusList
			if m.isSplitView {
				m.focused = focusDetail
			} else {
				m.showDetails = true
				m.focused = focusDetail
			}
			m.updateViewportContent()
		}
	case "y":
		// Copy selected commit SHA to clipboard
		if commit := m.historyView.SelectedCommit(); commit != nil {
			if err := clipboard.WriteAll(commit.SHA); err != nil {
				m.statusMsg = fmt.Sprintf("❌ Clipboard error: %v", err)
				m.statusIsError = true
			} else {
				m.statusMsg = fmt.Sprintf("📋 Copied %s to clipboard", commit.ShortSHA)
				m.statusIsError = false
			}
		} else {
			m.statusMsg = "❌ No commit selected"
			m.statusIsError = true
		}
	case "c":
		// Cycle confidence threshold
		m.historyView.CycleConfidence()
		conf := m.historyView.GetMinConfidence()
		if conf == 0 {
			m.statusMsg = "🔍 Showing all commits"
		} else {
			m.statusMsg = fmt.Sprintf("🔍 Confidence filter: ≥%.0f%%", conf*100)
		}
		m.statusIsError = false
	case "/":
		// Search hint - actual search would require text input
		m.statusMsg = "💡 Use 'f' for author filter, 'c' for confidence filter"
		m.statusIsError = false
	case "f":
		// Toggle author filter (simple toggle for now)
		m.statusMsg = "💡 Author filter: Use 'c' to cycle confidence thresholds"
		m.statusIsError = false
	case "H", "esc":
		// Exit history view
		m.isHistoryView = false
		m.focused = focusList
	}
	return m
}

// handleRecipePickerKeys handles keyboard input when recipe picker is focused
func (m Model) handleRecipePickerKeys(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "j", "down":
		m.recipePicker.MoveDown()
	case "k", "up":
		m.recipePicker.MoveUp()
	case "esc":
		m.showRecipePicker = false
		m.focused = focusList
	case "enter":
		// Apply selected recipe
		if selected := m.recipePicker.SelectedRecipe(); selected != nil {
			m.activeRecipe = selected
			m.applyRecipe(selected)
		}
		m.showRecipePicker = false
		m.focused = focusList
	}
	return m
}

// handleRepoPickerKeys handles keyboard input when repo picker is focused (workspace mode).
func (m Model) handleRepoPickerKeys(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "j", "down":
		m.repoPicker.MoveDown()
	case "k", "up":
		m.repoPicker.MoveUp()
	case " ", "space":
		m.repoPicker.ToggleSelected()
	case "a":
		m.repoPicker.SelectAll()
	case "esc", "q":
		m.showRepoPicker = false
		m.focused = focusList
	case "enter":
		selected := m.repoPicker.SelectedRepos()

		// Normalize: nil means "all repos" (no filter). Also treat empty as "all" to avoid hiding everything.
		if len(selected) == 0 || len(selected) == len(m.availableRepos) {
			m.activeRepos = nil
			m.statusMsg = "Repo filter: all repos"
		} else {
			m.activeRepos = selected
			m.statusMsg = fmt.Sprintf("Repo filter: %s", formatRepoList(sortedRepoKeys(selected), 3))
		}
		m.statusIsError = false

		// Apply filter to views
		if m.activeRecipe != nil {
			m.applyRecipe(m.activeRecipe)
		} else {
			m.applyFilter()
		}

		m.showRepoPicker = false
		m.focused = focusList
	}
	return m
}

// handleLabelPickerKeys handles keyboard input when label picker is focused (bv-126)
func (m Model) handleLabelPickerKeys(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "esc":
		m.showLabelPicker = false
		m.focused = focusList
	case "j", "down", "ctrl+n":
		m.labelPicker.MoveDown()
	case "k", "up", "ctrl+p":
		m.labelPicker.MoveUp()
	case "enter":
		if selected := m.labelPicker.SelectedLabel(); selected != "" {
			m.currentFilter = "label:" + selected
			m.applyFilter()
			m.statusMsg = fmt.Sprintf("Filtered by label: %s", selected)
			m.statusIsError = false
		}
		m.showLabelPicker = false
		m.focused = focusList
	default:
		// Pass other keys to text input for fuzzy search
		m.labelPicker.UpdateInput(msg)
	}
	return m
}

// handleInsightsKeys handles keyboard input when insights panel is focused
func (m Model) handleInsightsKeys(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "esc":
		m.focused = focusList
	case "j", "down":
		m.insightsPanel.MoveDown()
	case "k", "up":
		m.insightsPanel.MoveUp()
	case "h", "left":
		m.insightsPanel.PrevPanel()
	case "l", "right", "tab":
		m.insightsPanel.NextPanel()
	case "e":
		// Toggle explanations
		m.insightsPanel.ToggleExplanations()
	case "x":
		// Toggle calculation details
		m.insightsPanel.ToggleCalculation()
	case "H":
		// Toggle heatmap view (bv-95)
		m.insightsPanel.ToggleHeatmap()
	case "enter":
		// Jump to selected issue in list view
		selectedID := m.insightsPanel.SelectedIssueID()
		if selectedID != "" {
			for i, item := range m.list.Items() {
				if issueItem, ok := item.(IssueItem); ok && issueItem.Issue.ID == selectedID {
					m.list.Select(i)
					break
				}
			}
			m.focused = focusList
			if m.isSplitView {
				m.focused = focusDetail
			} else {
				m.showDetails = true
				m.focused = focusDetail
			}
			m.updateViewportContent()
		}
	}
	return m
}

// handleListKeys handles keyboard input when the list is focused
func (m Model) handleListKeys(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "enter":
		if m.isSplitView {
			// In split view, update the detail pane for the current selection
			m.updateViewportContent()
		} else {
			// In non-split view, open detail view
			m.showDetails = true
			m.focused = focusDetail
			m.updateViewportContent()
		}
	case "home":
		m.list.Select(0)
	case "G", "end":
		if len(m.list.Items()) > 0 {
			m.list.Select(len(m.list.Items()) - 1)
		}
	case "ctrl+d":
		// Page down
		itemCount := len(m.list.Items())
		if itemCount > 0 {
			currentIdx := m.list.Index()
			newIdx := currentIdx + m.height/3
			if newIdx >= itemCount {
				newIdx = itemCount - 1
			}
			m.list.Select(newIdx)
		}
	case "ctrl+u":
		// Page up
		if len(m.list.Items()) > 0 {
			currentIdx := m.list.Index()
			newIdx := currentIdx - m.height/3
			if newIdx < 0 {
				newIdx = 0
			}
			m.list.Select(newIdx)
		}
	case "o":
		m.currentFilter = "open"
		m.applyFilter()
	case "c":
		m.currentFilter = "closed"
		m.applyFilter()
	case "r":
		m.currentFilter = "ready"
		m.applyFilter()
	case "a":
		m.currentFilter = "all"
		m.applyFilter()
	case "t":
		// Toggle time-travel mode off, or show prompt for custom revision
		if m.timeTravelMode {
			m.exitTimeTravelMode()
		} else {
			// Show input prompt for revision
			m.showTimeTravelPrompt = true
			m.timeTravelInput.SetValue("")
			m.timeTravelInput.Focus()
			m.focused = focusTimeTravelInput
		}
	case "T":
		// Quick time-travel with default HEAD~5
		if m.timeTravelMode {
			m.exitTimeTravelMode()
		} else {
			m.enterTimeTravelMode("HEAD~5")
		}
	case "C":
		// Copy selected issue to clipboard
		m.copyIssueToClipboard()
	case "O":
		// Open beads.jsonl in editor
		m.openInEditor()
	case "H":
		// Toggle history view
		if !m.isHistoryView {
			m.enterHistoryView()
		}
	case "S":
		// Apply triage recipe - sort by triage score (bv-151)
		if r := m.recipeLoader.Get("triage"); r != nil {
			m.activeRecipe = r
			m.applyRecipe(r)
		}
	}
	return m
}

// handleTimeTravelInputKeys handles keyboard input for the time-travel revision prompt
func (m Model) handleTimeTravelInputKeys(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "enter":
		// Submit the revision
		revision := strings.TrimSpace(m.timeTravelInput.Value())
		if revision == "" {
			revision = "HEAD~5" // Default if empty
		}
		m.showTimeTravelPrompt = false
		m.timeTravelInput.Blur()
		m.focused = focusList
		m.enterTimeTravelMode(revision)
	case "esc":
		// Cancel
		m.showTimeTravelPrompt = false
		m.timeTravelInput.Blur()
		m.focused = focusList
	default:
		// Update the textinput
		m.timeTravelInput, _ = m.timeTravelInput.Update(msg)
	}
	return m
}

// handleHelpKeys handles keyboard input when the help overlay is focused
func (m Model) handleHelpKeys(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "j", "down":
		m.helpScroll++
	case "k", "up":
		if m.helpScroll > 0 {
			m.helpScroll--
		}
	case "ctrl+d":
		m.helpScroll += 10
	case "ctrl+u":
		m.helpScroll -= 10
		if m.helpScroll < 0 {
			m.helpScroll = 0
		}
	case "home", "g":
		m.helpScroll = 0
	case "G", "end":
		// Will be clamped in render
		m.helpScroll = 999
	case "q", "esc", "?", "f1":
		// Close help overlay
		m.showHelp = false
		m.helpScroll = 0
		m.focused = focusList
	default:
		// Any other key dismisses help
		m.showHelp = false
		m.helpScroll = 0
		m.focused = focusList
	}
	return m
}
