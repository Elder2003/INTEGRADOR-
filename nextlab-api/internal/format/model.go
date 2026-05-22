package format

import (
	"time"

	"github.com/google/uuid"
)

type FormatReport struct {
	ID           uuid.UUID      `json:"id"`
	SubmissionID *uuid.UUID     `json:"submission_id,omitempty"`
	SessionID    uuid.UUID      `json:"session_id"`
	ParseStatus  string         `json:"parse_status"`
	FailReason   string         `json:"fail_reason,omitempty"`
	FileType     string         `json:"file_type,omitempty"`
	PageCount    int            `json:"page_count"`
	WordCount    int            `json:"word_count"`
	CharCount    int            `json:"char_count"`
	DocMetadata  map[string]any `json:"doc_metadata,omitempty"`
	Compliant    bool           `json:"compliant"`
	RuleResults  []RuleResult   `json:"rule_results"`
	CreatedAt    time.Time      `json:"created_at"`
}

type RuleResult struct {
	Rule     string `json:"rule"`
	Status   string `json:"status"`
	Actual   any    `json:"actual"`
	Required any    `json:"required"`
}

type FormatRulesConfig struct {
	MinPages      *int     `json:"min_pages,omitempty"`
	MaxPages      *int     `json:"max_pages,omitempty"`
	MinWordCount  *int     `json:"min_word_count,omitempty"`
	RequiredTypes []string `json:"required_types,omitempty"`
}

// FileInfo holds extracted metadata from a parsed file.
type FileInfo struct {
	FileType    string
	PageCount   int
	WordCount   int
	CharCount   int
	DocMetadata map[string]any
}

// ParseError is returned by ParseFile for known, non-fatal parse failures.
type ParseError struct{ Reason string }

func (e *ParseError) Error() string { return "format_parse_error: " + e.Reason }
