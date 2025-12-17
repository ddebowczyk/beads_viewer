package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Dicklesworthstone/beads_viewer/pkg/export"
	"github.com/Dicklesworthstone/beads_viewer/pkg/loader"

	"github.com/atotto/clipboard"
)

// exportToMarkdown exports all issues to a Markdown file with auto-generated filename
func (m *Model) exportToMarkdown() {
	// Generate smart filename: beads_report_<project>_YYYY-MM-DD.md
	filename := m.generateExportFilename()

	// Export the issues
	err := export.SaveMarkdownToFile(m.issues, filename)
	if err != nil {
		m.statusMsg = fmt.Sprintf("❌ Export failed: %v", err)
		m.statusIsError = true
		return
	}

	m.statusMsg = fmt.Sprintf("✅ Exported %d issues to %s", len(m.issues), filename)
	m.statusIsError = false
}

// generateExportFilename creates a smart filename based on project and date
func (m *Model) generateExportFilename() string {
	// Get project name from current directory
	projectName := "beads"
	if cwd, err := os.Getwd(); err == nil {
		projectName = filepath.Base(cwd)
		// Sanitize: replace spaces and special chars with underscores
		projectName = strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
				return r
			}
			return '_'
		}, projectName)
	}

	// Format: beads_report_<project>_YYYY-MM-DD.md
	timestamp := time.Now().Format("2006-01-02")
	return fmt.Sprintf("beads_report_%s_%s.md", projectName, timestamp)
}

// copyIssueToClipboard copies the selected issue to clipboard as Markdown
func (m *Model) copyIssueToClipboard() {
	selectedItem := m.list.SelectedItem()
	if selectedItem == nil {
		m.statusMsg = "❌ No issue selected"
		m.statusIsError = true
		return
	}

	issueItem, ok := selectedItem.(IssueItem)
	if !ok {
		m.statusMsg = "❌ Invalid item type"
		m.statusIsError = true
		return
	}
	issue := issueItem.Issue

	// Format issue as Markdown
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s %s\n\n", GetTypeIconMD(string(issue.IssueType)), issue.Title))
	sb.WriteString(fmt.Sprintf("**ID:** %s  \n", issue.ID))
	sb.WriteString(fmt.Sprintf("**Status:** %s  \n", strings.ToUpper(string(issue.Status))))
	sb.WriteString(fmt.Sprintf("**Priority:** P%d  \n", issue.Priority))
	if issue.Assignee != "" {
		sb.WriteString(fmt.Sprintf("**Assignee:** @%s  \n", issue.Assignee))
	}
	sb.WriteString(fmt.Sprintf("**Created:** %s  \n", issue.CreatedAt.Format("2006-01-02")))

	if len(issue.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("**Labels:** %s  \n", strings.Join(issue.Labels, ", ")))
	}

	if issue.Description != "" {
		sb.WriteString(fmt.Sprintf("\n## Description\n\n%s\n", issue.Description))
	}

	if issue.AcceptanceCriteria != "" {
		sb.WriteString(fmt.Sprintf("\n## Acceptance Criteria\n\n%s\n", issue.AcceptanceCriteria))
	}

	// Dependencies
	if len(issue.Dependencies) > 0 {
		sb.WriteString("\n## Dependencies\n\n")
		for _, dep := range issue.Dependencies {
			if dep == nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", dep.DependsOnID, dep.Type))
		}
	}

	// Copy to clipboard
	err := clipboard.WriteAll(sb.String())
	if err != nil {
		m.statusMsg = fmt.Sprintf("❌ Clipboard error: %v", err)
		m.statusIsError = true
		return
	}

	m.statusMsg = fmt.Sprintf("📋 Copied %s to clipboard", issue.ID)
	m.statusIsError = false
}

// openInEditor opens the beads file in the user's preferred editor
// Uses m.beadsPath which respects issues.jsonl (canonical per beads upstream)
func (m *Model) openInEditor() {
	// Use the configured beadsPath instead of hardcoded path
	beadsFile := m.beadsPath
	if beadsFile == "" {
		cwd, _ := os.Getwd()
		if found, err := loader.FindJSONLPath(filepath.Join(cwd, ".beads")); err == nil {
			beadsFile = found
		}
	}
	if beadsFile == "" {
		m.statusMsg = "❌ No .beads directory or beads.jsonl found"
		m.statusIsError = true
		return
	}
	if _, err := os.Stat(beadsFile); os.IsNotExist(err) {
		m.statusMsg = fmt.Sprintf("❌ Beads file not found: %s", beadsFile)
		m.statusIsError = true
		return
	}

	// Determine editor - prefer GUI editors that work in background
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}

	// Check if it's a terminal editor (won't work well with TUI)
	terminalEditors := map[string]bool{
		"vim": true, "vi": true, "nvim": true, "nano": true,
		"emacs": true, "pico": true, "joe": true, "ne": true,
	}
	editorBase := filepath.Base(editor)
	if terminalEditors[editorBase] {
		m.statusMsg = fmt.Sprintf("⚠️ %s is a terminal editor - set $EDITOR to a GUI editor or quit first", editorBase)
		m.statusIsError = true
		return
	}

	// If no editor set, try platform-specific GUI options
	if editor == "" {
		switch runtime.GOOS {
		case "darwin":
			// Use 'open' to launch default app for .jsonl files
			cmd := exec.Command("open", "-t", beadsFile)
			if err := cmd.Start(); err == nil {
				m.statusMsg = "📝 Opened in default text editor"
				m.statusIsError = false
				return
			}
		case "windows":
			editor = "notepad"
		case "linux":
			// Try xdg-open first, then common GUI editors
			for _, tryEditor := range []string{"xdg-open", "code", "gedit", "kate", "xed"} {
				if _, err := exec.LookPath(tryEditor); err == nil {
					editor = tryEditor
					break
				}
			}
		}
	}

	if editor == "" {
		m.statusMsg = "❌ No GUI editor found. Set $EDITOR to a GUI editor"
		m.statusIsError = true
		return
	}

	// Launch GUI editor in background
	cmd := exec.Command(editor, beadsFile)
	if err := cmd.Start(); err != nil {
		m.statusMsg = fmt.Sprintf("❌ Failed to open editor: %v", err)
		m.statusIsError = true
		return
	}

	m.statusMsg = fmt.Sprintf("📝 Opened in %s", filepath.Base(editor))
	m.statusIsError = false
}
