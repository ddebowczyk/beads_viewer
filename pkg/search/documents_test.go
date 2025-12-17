package search

import (
	"testing"

	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
)

// =============================================================================
// IssueDocument Tests
// =============================================================================

func TestIssueDocument(t *testing.T) {
	tests := []struct {
		name        string
		issue       model.Issue
		expected    string
	}{
		{
			name: "title and description",
			issue: model.Issue{
				Title:       "Fix login bug",
				Description: "Users cannot log in on mobile",
			},
			expected: "Fix login bug\nUsers cannot log in on mobile",
		},
		{
			name: "title only",
			issue: model.Issue{
				Title:       "Add dark mode",
				Description: "",
			},
			expected: "Add dark mode",
		},
		{
			name: "description only",
			issue: model.Issue{
				Title:       "",
				Description: "Performance improvements needed",
			},
			expected: "Performance improvements needed",
		},
		{
			name: "both empty",
			issue: model.Issue{
				Title:       "",
				Description: "",
			},
			expected: "",
		},
		{
			name: "title with whitespace",
			issue: model.Issue{
				Title:       "  Trimmed title  ",
				Description: "Some description",
			},
			expected: "Trimmed title\nSome description",
		},
		{
			name: "description with whitespace",
			issue: model.Issue{
				Title:       "Some title",
				Description: "  Trimmed description  ",
			},
			expected: "Some title\nTrimmed description",
		},
		{
			name: "both have whitespace",
			issue: model.Issue{
				Title:       "  Title  ",
				Description: "  Description  ",
			},
			expected: "Title\nDescription",
		},
		{
			name: "whitespace-only title treated as empty",
			issue: model.Issue{
				Title:       "   ",
				Description: "Actual description",
			},
			expected: "Actual description",
		},
		{
			name: "whitespace-only description treated as empty",
			issue: model.Issue{
				Title:       "Actual title",
				Description: "   ",
			},
			expected: "Actual title",
		},
		{
			name: "multiline description",
			issue: model.Issue{
				Title: "Feature request",
				Description: `Line 1
Line 2
Line 3`,
			},
			expected: "Feature request\nLine 1\nLine 2\nLine 3",
		},
		{
			name: "unicode content",
			issue: model.Issue{
				Title:       "日本語タイトル",
				Description: "Description in English",
			},
			expected: "日本語タイトル\nDescription in English",
		},
		{
			name: "newlines in title preserved",
			issue: model.Issue{
				Title:       "Title\nwith\nnewlines",
				Description: "Desc",
			},
			expected: "Title\nwith\nnewlines\nDesc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IssueDocument(tt.issue)
			if result != tt.expected {
				t.Errorf("IssueDocument() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// =============================================================================
// DocumentsFromIssues Tests
// =============================================================================

func TestDocumentsFromIssues(t *testing.T) {
	tests := []struct {
		name     string
		issues   []model.Issue
		expected map[string]string
	}{
		{
			name:     "nil issues",
			issues:   nil,
			expected: map[string]string{},
		},
		{
			name:     "empty issues",
			issues:   []model.Issue{},
			expected: map[string]string{},
		},
		{
			name: "single issue",
			issues: []model.Issue{
				{ID: "issue-1", Title: "Bug fix", Description: "Fix the bug"},
			},
			expected: map[string]string{
				"issue-1": "Bug fix\nFix the bug",
			},
		},
		{
			name: "multiple issues",
			issues: []model.Issue{
				{ID: "issue-1", Title: "Bug 1", Description: "Desc 1"},
				{ID: "issue-2", Title: "Bug 2", Description: "Desc 2"},
				{ID: "issue-3", Title: "Bug 3", Description: "Desc 3"},
			},
			expected: map[string]string{
				"issue-1": "Bug 1\nDesc 1",
				"issue-2": "Bug 2\nDesc 2",
				"issue-3": "Bug 3\nDesc 3",
			},
		},
		{
			name: "skips issues with empty ID",
			issues: []model.Issue{
				{ID: "issue-1", Title: "Valid", Description: "Valid desc"},
				{ID: "", Title: "Invalid", Description: "No ID"},
				{ID: "issue-2", Title: "Valid 2", Description: "Valid desc 2"},
			},
			expected: map[string]string{
				"issue-1": "Valid\nValid desc",
				"issue-2": "Valid 2\nValid desc 2",
			},
		},
		{
			name: "all issues have empty IDs",
			issues: []model.Issue{
				{ID: "", Title: "No ID 1", Description: "Desc 1"},
				{ID: "", Title: "No ID 2", Description: "Desc 2"},
			},
			expected: map[string]string{},
		},
		{
			name: "duplicate IDs last wins",
			issues: []model.Issue{
				{ID: "dupe", Title: "First", Description: "First desc"},
				{ID: "dupe", Title: "Second", Description: "Second desc"},
			},
			expected: map[string]string{
				"dupe": "Second\nSecond desc",
			},
		},
		{
			name: "issue with only title",
			issues: []model.Issue{
				{ID: "title-only", Title: "Just a title", Description: ""},
			},
			expected: map[string]string{
				"title-only": "Just a title",
			},
		},
		{
			name: "issue with only description",
			issues: []model.Issue{
				{ID: "desc-only", Title: "", Description: "Just a description"},
			},
			expected: map[string]string{
				"desc-only": "Just a description",
			},
		},
		{
			name: "issue with empty title and description",
			issues: []model.Issue{
				{ID: "empty-content", Title: "", Description: ""},
			},
			expected: map[string]string{
				"empty-content": "",
			},
		},
		{
			name: "mixed content issues",
			issues: []model.Issue{
				{ID: "full", Title: "Full title", Description: "Full description"},
				{ID: "title-only", Title: "Title only", Description: ""},
				{ID: "desc-only", Title: "", Description: "Description only"},
				{ID: "empty", Title: "", Description: ""},
			},
			expected: map[string]string{
				"full":       "Full title\nFull description",
				"title-only": "Title only",
				"desc-only":  "Description only",
				"empty":      "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DocumentsFromIssues(tt.issues)

			if len(result) != len(tt.expected) {
				t.Errorf("DocumentsFromIssues() returned %d docs, want %d", len(result), len(tt.expected))
			}

			for id, expectedDoc := range tt.expected {
				actualDoc, ok := result[id]
				if !ok {
					t.Errorf("Missing expected ID %q in result", id)
					continue
				}
				if actualDoc != expectedDoc {
					t.Errorf("DocumentsFromIssues()[%q] = %q, want %q", id, actualDoc, expectedDoc)
				}
			}

			// Check for unexpected IDs
			for id := range result {
				if _, expected := tt.expected[id]; !expected {
					t.Errorf("Unexpected ID %q in result", id)
				}
			}
		})
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestDocumentsFromIssues_LargeDataset(t *testing.T) {
	// Test with a reasonably large dataset to ensure no performance issues
	issues := make([]model.Issue, 1000)
	for i := range issues {
		issues[i] = model.Issue{
			ID:          "issue-" + string(rune('A'+i%26)) + string(rune('0'+i%10)),
			Title:       "Test Issue Title",
			Description: "Test issue description for indexing",
		}
	}

	result := DocumentsFromIssues(issues)

	// Should have unique IDs (duplicates overwritten)
	if len(result) == 0 {
		t.Error("Expected non-empty result for large dataset")
	}

	// Spot check a few entries
	for _, doc := range result {
		if doc == "" {
			t.Error("Found empty document in result")
		}
	}
}

func TestIssueDocument_PreservesContent(t *testing.T) {
	// Ensure content is preserved exactly (except whitespace trimming)
	issue := model.Issue{
		Title:       "Special chars: <>&\"'",
		Description: "Code: `fmt.Println()` and more",
	}

	result := IssueDocument(issue)
	expected := "Special chars: <>&\"'\nCode: `fmt.Println()` and more"

	if result != expected {
		t.Errorf("Content not preserved correctly:\ngot: %q\nwant: %q", result, expected)
	}
}
