package projects

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Elder2003/nextlab-api/internal/ai"
	"github.com/google/uuid"
)

type GitAnalysisService struct {
	repo  *Repository
	aiSvc *ai.Service
}

func newGitAnalysisService(repo *Repository, aiSvc *ai.Service) *GitAnalysisService {
	return &GitAnalysisService{repo: repo, aiSvc: aiSvc}
}

// SubmitGitURL records a student's repo URL and enqueues an analysis job.
func (s *GitAnalysisService) SubmitGitURL(ctx context.Context, studentID, actividadID uuid.UUID, repoURL string) (*GitSubmission, string, error) {
	// Validate URL is parseable
	if _, err := url.ParseRequestURI(repoURL); err != nil {
		return nil, "", fmt.Errorf("invalid repo_url")
	}

	_, cursoID, _, err := s.repo.GetActividadTipo(ctx, actividadID)
	if err != nil {
		return nil, "", err
	}
	enrolled, err := s.repo.StudentEnrolledInCurso(ctx, studentID, cursoID)
	if err != nil {
		return nil, "", err
	}
	if !enrolled {
		return nil, "", ErrForbidden
	}

	sub, err := s.repo.UpsertGitSubmission(ctx, actividadID, studentID, repoURL)
	if err != nil {
		return nil, "", fmt.Errorf("upsert git submission: %w", err)
	}

	analysis, err := s.repo.InsertGitAnalysis(ctx, sub.ID)
	if err != nil {
		return nil, "", fmt.Errorf("insert git analysis: %w", err)
	}

	if err := s.repo.InsertGitAnalysisJob(ctx, analysis.ID); err != nil {
		return nil, "", fmt.Errorf("insert git analysis job: %w", err)
	}

	return sub, analysis.Status, nil
}

// GetGitReport returns the analysis report for a student's submission; students get 403.
func (s *GitAnalysisService) GetGitReport(ctx context.Context, callerID uuid.UUID, callerRole string, actividadID, studentID uuid.UUID) (*GitReportResponse, error) {
	if callerRole == "student" {
		return nil, ErrForbidden
	}

	sub, err := s.repo.GetGitSubmission(ctx, actividadID, studentID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrNotFound
	}

	analysis, err := s.repo.GetLatestGitAnalysis(ctx, sub.ID)
	if err != nil {
		return nil, err
	}

	resp := &GitReportResponse{
		GitSubmissionID: sub.ID,
		StudentID:       studentID,
		RepoURL:         sub.RepoURL,
	}
	if analysis != nil {
		resp.Analysis = analysis
		resp.AnalysisStatus = analysis.Status
	} else {
		resp.AnalysisStatus = "not_started"
	}
	return resp, nil
}

// GradeGitAnalysis stores a final grade on the latest analysis record.
func (s *GitAnalysisService) GradeGitAnalysis(ctx context.Context, docenteID, actividadID, studentID uuid.UUID, grade float64, overrideReason string) (*GitAnalysis, error) {
	if grade < 0 || grade > 100 {
		return nil, fmt.Errorf("grade must be 0–100")
	}

	sub, err := s.repo.GetGitSubmission(ctx, actividadID, studentID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrNotFound
	}

	analysis, err := s.repo.GetLatestGitAnalysis(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	if analysis == nil || analysis.Status == "pending" {
		return nil, ErrNotFound
	}

	if err := s.repo.GradeGitAnalysis(ctx, analysis.ID, docenteID, grade, overrideReason); err != nil {
		return nil, err
	}

	return s.repo.GetLatestGitAnalysis(ctx, sub.ID)
}
