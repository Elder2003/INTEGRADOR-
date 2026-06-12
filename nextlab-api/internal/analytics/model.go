package analytics

import (
	"github.com/google/uuid"
)

// M17: Percentiles (US2)

type PercentilesResult struct {
	ActividadID      uuid.UUID `json:"actividad_id"`
	StudentID        uuid.UUID `json:"student_id"`
	StudentScore     float64   `json:"student_score"`
	StudentPercentile float64  `json:"student_percentile"`
	P25              float64   `json:"p25_score"`
	P50              float64   `json:"p50_score"`
	P75              float64   `json:"p75_score"`
	ClassSize        int       `json:"class_size"`
}

// M17: Admin Analytics (US4)

type AdminOverview struct {
	Period             string `json:"period"`
	TotalSessions      int    `json:"total_sessions"`
	TotalSubmissions   int    `json:"total_submissions"`
	ActiveDetections   int    `json:"active_detections"`
	ActiveStudents     int    `json:"active_students"`
	ActiveCourses      int    `json:"active_courses"`
}

type CarreraIntegrity struct {
	Carrera          string  `json:"carrera"`
	SessionsAnalyzed int     `json:"sessions_analyzed"`
	SessionsFlagged  int     `json:"sessions_flagged"`
	FlagRatePct      float64 `json:"flag_rate_pct"`
	AvgRiskScore     float64 `json:"avg_risk_score"`
}

type MateriaSubmissions struct {
	CodigoMateria    string   `json:"codigo_materia"`
	MateriaNombre    string   `json:"materia_name"`
	OpenActivities   int      `json:"open_activities"`
	Received         int      `json:"submissions_received"`
	SubmissionRatePct float64 `json:"submission_rate_pct"`
	AvgGrade         *float64 `json:"avg_grade"`
}

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
