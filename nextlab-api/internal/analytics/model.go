package analytics

import "github.com/google/uuid"

type GradeDistribution struct {
	Range0_49   int `json:"range_0_49"`
	Range50_69  int `json:"range_50_69"`
	Range70_89  int `json:"range_70_89"`
	Range90_100 int `json:"range_90_100"`
}

type ActivityAnalytics struct {
	ActividadID       uuid.UUID         `json:"actividad_id"`
	SubmissionCount   int               `json:"submission_count"`
	PendingCount      int               `json:"pending_count"`
	LateCount         int               `json:"late_count"`
	FlaggedCount      int               `json:"flagged_count"`
	GradeDistribution GradeDistribution `json:"grade_distribution"`
	AvgGrade          *float64          `json:"avg_grade"`
}

type CourseAnalytics struct {
	CursoID          uuid.UUID           `json:"curso_id"`
	Activities       []ActivityAnalytics `json:"activities"`
	TotalSubmissions int                 `json:"total_submissions"`
	AvgGrade         *float64            `json:"avg_grade"`
	FlagRate         float64             `json:"flag_rate"`
}
