package format

import "testing"

func TestEvaluateRulesReturnsFailuresForInvalidSubmission(t *testing.T) {
	minPages := 2
	maxPages := 10
	minWords := 300
	rules := &FormatRulesConfig{
		MinPages:      &minPages,
		MaxPages:      &maxPages,
		MinWordCount:  &minWords,
		RequiredTypes: []string{"pdf"},
	}
	info := FileInfo{
		FileType:  "docx",
		PageCount: 1,
		WordCount: 120,
	}

	results, compliant := EvaluateRules(info, rules)

	if compliant {
		t.Fatal("compliant = true, want false")
	}
	if len(results) != 4 {
		t.Fatalf("rule results = %d, want 4", len(results))
	}
	wantStatuses := map[string]string{
		"min_pages":      "fail",
		"max_pages":      "pass",
		"min_word_count": "fail",
		"file_type":      "fail",
	}
	for _, result := range results {
		if result.Status != wantStatuses[result.Rule] {
			t.Fatalf("%s status = %q, want %q", result.Rule, result.Status, wantStatuses[result.Rule])
		}
	}
}
