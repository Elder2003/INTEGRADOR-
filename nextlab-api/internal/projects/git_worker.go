package projects

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Elder2003/nextlab-api/internal/ai"
	"github.com/Elder2003/nextlab-api/internal/alerts"
	gogit "github.com/go-git/go-git/v5"
	gogitcfg "github.com/go-git/go-git/v5/plumbing/transport/http"
)

const cloneDepth = 200

// ProcessGitAnalysisJob clones the repo, extracts commit stats, runs heuristics, and dispatches to the LLM.
// Clone or extraction failures are stored as failed analysis records — the submission is never affected (FR-012, Principle VII).
func ProcessGitAnalysisJob(ctx context.Context, job *GitAnalysisJob, repo *Repository, aiSvc *ai.Service, alertWriter alerts.AlertWriter) error {
	slog.InfoContext(ctx, "processing git analysis job",
		"job_id", job.ID, "git_analysis_id", job.GitAnalysisID, "repo_url", job.RepoURL)

	_, _, deadline, err := repo.GetActividadTipo(ctx, job.ActividadID)
	if err != nil {
		slog.WarnContext(ctx, "could not fetch actividad deadline; deadline_rush will be 0", "err", err)
		deadline = time.Time{}
	}

	// ── Phase 1: Clone ────────────────────────────────────────────────────────
	dir, cleanup, cloneErr := cloneRepo(ctx, job.RepoURL)
	if cloneErr != nil {
		slog.WarnContext(ctx, "clone failed; storing clone_error", "job_id", job.ID, "err", cloneErr)
		_ = repo.UpdateGitAnalysisFailed(ctx, job.GitAnalysisID, "clone_error")
		return nil
	}
	defer cleanup()

	// ── Phase 2: Extract commit stats ─────────────────────────────────────────
	stats, statsErr := extractCommitStats(dir, deadline)
	if statsErr != nil {
		slog.WarnContext(ctx, "commit extraction failed", "job_id", job.ID, "err", statsErr)
		_ = repo.UpdateGitAnalysisFailed(ctx, job.GitAnalysisID, "clone_error")
		return nil
	}

	if err := repo.UpdateGitAnalysisSuccess(ctx, job.GitAnalysisID,
		stats.commitCount, stats.authors, stats.firstCommitAt, stats.lastCommitAt,
		stats.deadlineRushPct, stats.consistencyScore, stats.flags,
	); err != nil {
		return fmt.Errorf("update git analysis success: %w", err)
	}

	// ── Phase 3: LLM advisory ─────────────────────────────────────────────────
	advisory, llmStatus := runGitLLM(ctx, aiSvc, stats)
	if err := repo.UpdateGitAnalysisLLM(ctx, job.GitAnalysisID, advisory, llmStatus); err != nil {
		slog.WarnContext(ctx, "failed to store llm result", "job_id", job.ID, "err", err)
	}

	slog.InfoContext(ctx, "git analysis complete",
		"job_id", job.ID, "commit_count", stats.commitCount, "flags", stats.flags, "llm_status", llmStatus)

	if alertWriter != nil && len(stats.flags) > 0 {
		if _, cursoID, _, lookupErr := repo.GetActividadTipo(ctx, job.ActividadID); lookupErr == nil {
			docenteID, _ := repo.GetCursoDocente(ctx, cursoID)
			aid := job.ActividadID
			_ = alertWriter.WriteAlert(ctx, alerts.Alert{
				DocenteID:   docenteID,
				ActividadID: &aid,
				Type:        alerts.AlertGitAnalysisDone,
				Advisory:    true,
				Payload:     map[string]any{"flag_count": len(stats.flags), "flags": stats.flags, "git_analysis_id": job.GitAnalysisID},
			})
		}
	}

	return nil
}

// ─── Clone ────────────────────────────────────────────────────────────────────

func cloneRepo(ctx context.Context, repoURL string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "gitanalysis-*")
	if err != nil {
		return "", nil, fmt.Errorf("mkdirtemp: %w", err)
	}
	cleanup := func() { os.RemoveAll(dir) }

	cloneURL, auth := extractURLAuth(repoURL)

	opts := &gogit.CloneOptions{
		URL:          cloneURL,
		Depth:        cloneDepth,
		SingleBranch: true,
		NoCheckout:   true,
	}
	if auth != nil {
		opts.Auth = auth
	}

	_, err = gogit.PlainCloneContext(ctx, dir, true, opts)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("clone: %w", err)
	}
	return dir, cleanup, nil
}

// extractURLAuth splits token@host URLs and returns a clean URL + BasicAuth.
func extractURLAuth(rawURL string) (string, *gogitcfg.BasicAuth) {
	u, err := url.Parse(rawURL)
	if err != nil || u.User == nil {
		return rawURL, nil
	}
	password, hasPassword := u.User.Password()
	if !hasPassword || password == "" {
		return rawURL, nil
	}
	auth := &gogitcfg.BasicAuth{
		Username: u.User.Username(),
		Password: password,
	}
	u.User = nil
	return u.String(), auth
}

// ─── Commit stats ─────────────────────────────────────────────────────────────

type commitStats struct {
	commitCount      int
	authors          []string
	firstCommitAt    *time.Time
	lastCommitAt     *time.Time
	deadlineRushPct  float64
	consistencyScore int
	flags            []string
}

func extractCommitStats(dir string, deadline time.Time) (*commitStats, error) {
	r, err := gogit.PlainOpen(dir)
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	ref, err := r.Head()
	if err != nil {
		// Empty repository — return zeroed stats (per spec edge case)
		return &commitStats{authors: []string{}, flags: []string{}}, nil
	}

	iter, err := r.Log(&gogit.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("log: %w", err)
	}
	defer iter.Close()

	authorSet := make(map[string]struct{})
	var first, last *time.Time
	total := 0
	rushCount := 0
	rushWindowStart := deadline.Add(-24 * time.Hour)

	for {
		c, iterErr := iter.Next()
		if iterErr != nil {
			break
		}
		total++

		name := strings.TrimSpace(c.Author.Name)
		if name == "" {
			name = strings.TrimSpace(c.Author.Email)
		}
		if name != "" {
			authorSet[name] = struct{}{}
		}

		when := c.Author.When
		if first == nil || when.Before(*first) {
			t := when
			first = &t
		}
		if last == nil || when.After(*last) {
			t := when
			last = &t
		}
		if !deadline.IsZero() && !when.Before(rushWindowStart) && !when.After(deadline) {
			rushCount++
		}
	}

	authors := make([]string, 0, len(authorSet))
	for a := range authorSet {
		authors = append(authors, a)
	}

	var deadlineRushPct float64
	if total > 0 && !deadline.IsZero() {
		deadlineRushPct = float64(rushCount) / float64(total) * 100
	}

	cs := 100 - int(deadlineRushPct)
	if cs < 0 {
		cs = 0
	}

	var flags []string
	if deadlineRushPct > 70 {
		flags = append(flags, "deadline_rush")
	}
	if len(authors) > 1 {
		flags = append(flags, "multiple_authors")
	}
	if flags == nil {
		flags = []string{}
	}

	return &commitStats{
		commitCount:      total,
		authors:          authors,
		firstCommitAt:    first,
		lastCommitAt:     last,
		deadlineRushPct:  deadlineRushPct,
		consistencyScore: cs,
		flags:            flags,
	}, nil
}

// ─── LLM advisory ─────────────────────────────────────────────────────────────

func runGitLLM(ctx context.Context, aiSvc *ai.Service, stats *commitStats) (advisory, status string) {
	if aiSvc == nil {
		return "", "failed"
	}

	systemMsg := "You are an academic integrity assistant reviewing a student's Git repository history. Provide a brief advisory assessment for the course instructor. Do NOT assign a final grade."
	userMsg := buildGitPrompt(stats)

	resp, _, err := aiSvc.Dispatch(ctx, ai.DispatchRequest{
		Messages: []ai.ChatMessage{
			{Role: "system", Content: systemMsg},
			{Role: "user", Content: userMsg},
		},
	}, 60)
	if err != nil || resp == nil || len(resp.Choices) == 0 {
		slog.WarnContext(ctx, "git analysis LLM dispatch failed", "err", err)
		return "", "failed"
	}

	raw := resp.Choices[0].Message.Content

	var parsed struct {
		AdvisoryText string `json:"advisory_text"`
	}
	if jsonErr := json.Unmarshal([]byte(raw), &parsed); jsonErr == nil && parsed.AdvisoryText != "" {
		return parsed.AdvisoryText, "success"
	}
	return raw, "success"
}

func buildGitPrompt(stats *commitStats) string {
	var sb strings.Builder
	sb.WriteString("## Git Repository Analysis\n\n")
	sb.WriteString(fmt.Sprintf("- Total commits: %d\n", stats.commitCount))
	sb.WriteString(fmt.Sprintf("- Distinct authors: %s\n", strings.Join(stats.authors, ", ")))
	if stats.firstCommitAt != nil {
		sb.WriteString(fmt.Sprintf("- First commit: %s\n", stats.firstCommitAt.Format(time.RFC3339)))
	}
	if stats.lastCommitAt != nil {
		sb.WriteString(fmt.Sprintf("- Last commit: %s\n", stats.lastCommitAt.Format(time.RFC3339)))
	}
	sb.WriteString(fmt.Sprintf("- Deadline-rush percentage (commits in last 24h before deadline): %.1f%%\n", stats.deadlineRushPct))
	sb.WriteString(fmt.Sprintf("- Consistency score: %d/100\n", stats.consistencyScore))
	if len(stats.flags) > 0 {
		sb.WriteString(fmt.Sprintf("- Heuristic flags raised: %s\n", strings.Join(stats.flags, ", ")))
	} else {
		sb.WriteString("- Heuristic flags raised: none\n")
	}
	sb.WriteString("\nProvide a brief advisory assessment of the commit history. ")
	sb.WriteString(`Format your response as JSON: {"advisory_text":"..."}`)
	return sb.String()
}

