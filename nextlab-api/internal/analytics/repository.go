package analytics

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ActivityAnalyticsQuery computes live aggregate statistics for a single activity.
// The query adapts to the activity tipo since grades live in different tables per tipo.
func (r *Repository) ActivityAnalyticsQuery(ctx context.Context, actividadID uuid.UUID, tipo string) (ActivityAnalytics, error) {
	out := ActivityAnalytics{ActividadID: actividadID}

	// Enrolled student count (denominator for pending)
	var enrolled int
	_ = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM inscripciones WHERE actividad_id = $1`, actividadID,
	).Scan(&enrolled)

	switch tipo {
	case "investigacion":
		err := r.db.QueryRow(ctx, `
			SELECT
				COUNT(DISTINCT ss.student_id)                                          AS submission_count,
				COUNT(DISTINCT ss.student_id) FILTER (WHERE ss.is_late = true)         AS late_count,
				COUNT(*) FILTER (WHERE ss.grade BETWEEN 0  AND 49)                    AS r0,
				COUNT(*) FILTER (WHERE ss.grade BETWEEN 50 AND 69)                    AS r50,
				COUNT(*) FILTER (WHERE ss.grade BETWEEN 70 AND 89)                    AS r70,
				COUNT(*) FILTER (WHERE ss.grade BETWEEN 90 AND 100)                   AS r90,
				AVG(ss.grade)                                                          AS avg_grade
			FROM stage_submissions ss
			JOIN investigacion_stages s ON ss.stage_id = s.id
			WHERE s.actividad_id = $1`,
			actividadID,
		).Scan(&out.SubmissionCount, &out.LateCount,
			&out.GradeDistribution.Range0_49, &out.GradeDistribution.Range50_69,
			&out.GradeDistribution.Range70_89, &out.GradeDistribution.Range90_100,
			&out.AvgGrade)
		if err != nil {
			return out, fmt.Errorf("investigacion analytics: %w", err)
		}

	case "git_analysis":
		err := r.db.QueryRow(ctx, `
			SELECT
				COUNT(*)                                                                AS submission_count,
				COUNT(*) FILTER (WHERE ga.grade BETWEEN 0  AND 49)                    AS r0,
				COUNT(*) FILTER (WHERE ga.grade BETWEEN 50 AND 69)                    AS r50,
				COUNT(*) FILTER (WHERE ga.grade BETWEEN 70 AND 89)                    AS r70,
				COUNT(*) FILTER (WHERE ga.grade BETWEEN 90 AND 100)                   AS r90,
				AVG(ga.grade)                                                          AS avg_grade
			FROM git_submissions gs
			LEFT JOIN git_analyses ga ON ga.actividad_id = gs.actividad_id AND ga.student_id = gs.student_id
			WHERE gs.actividad_id = $1`,
			actividadID,
		).Scan(&out.SubmissionCount,
			&out.GradeDistribution.Range0_49, &out.GradeDistribution.Range50_69,
			&out.GradeDistribution.Range70_89, &out.GradeDistribution.Range90_100,
			&out.AvgGrade)
		if err != nil {
			return out, fmt.Errorf("git_analysis analytics: %w", err)
		}

	case "defensa_oral":
		err := r.db.QueryRow(ctx, `
			SELECT
				COUNT(*) FILTER (WHERE scheduled_at IS NOT NULL)                       AS submission_count,
				COUNT(*) FILTER (WHERE grade BETWEEN 0  AND 49)                       AS r0,
				COUNT(*) FILTER (WHERE grade BETWEEN 50 AND 69)                       AS r50,
				COUNT(*) FILTER (WHERE grade BETWEEN 70 AND 89)                       AS r70,
				COUNT(*) FILTER (WHERE grade BETWEEN 90 AND 100)                      AS r90,
				AVG(grade)                                                             AS avg_grade
			FROM defense_appointments
			WHERE actividad_id = $1`,
			actividadID,
		).Scan(&out.SubmissionCount,
			&out.GradeDistribution.Range0_49, &out.GradeDistribution.Range50_69,
			&out.GradeDistribution.Range70_89, &out.GradeDistribution.Range90_100,
			&out.AvgGrade)
		if err != nil {
			return out, fmt.Errorf("defensa_oral analytics: %w", err)
		}

	default: // tarea, informe, examen, practica (grades live in grade_decisions via exam_sessions)
		err := r.db.QueryRow(ctx, `
			SELECT
				COUNT(sub.id)                                                              AS submission_count,
				COUNT(sub.id) FILTER (WHERE sub.is_late = true)                           AS late_count,
				COUNT(*) FILTER (WHERE gd.final_grade BETWEEN 0  AND 49)                 AS r0,
				COUNT(*) FILTER (WHERE gd.final_grade BETWEEN 50 AND 69)                 AS r50,
				COUNT(*) FILTER (WHERE gd.final_grade BETWEEN 70 AND 89)                 AS r70,
				COUNT(*) FILTER (WHERE gd.final_grade BETWEEN 90 AND 100)                AS r90,
				AVG(gd.final_grade)                                                        AS avg_grade
			FROM submissions sub
			LEFT JOIN exam_sessions es ON es.actividad_id = sub.actividad_id AND es.student_id = sub.student_id
			LEFT JOIN grade_decisions gd ON gd.session_id = es.id
			WHERE sub.actividad_id = $1`,
			actividadID,
		).Scan(&out.SubmissionCount, &out.LateCount,
			&out.GradeDistribution.Range0_49, &out.GradeDistribution.Range50_69,
			&out.GradeDistribution.Range70_89, &out.GradeDistribution.Range90_100,
			&out.AvgGrade)
		if err != nil {
			return out, fmt.Errorf("default analytics: %w", err)
		}
	}

	out.PendingCount = enrolled - out.SubmissionCount
	if out.PendingCount < 0 {
		out.PendingCount = 0
	}

	// Flagged count from alerts table
	_ = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM alerts
		 WHERE actividad_id = $1 AND type IN ('integrity_flag','format_violation')`,
		actividadID,
	).Scan(&out.FlaggedCount)

	return out, nil
}

// CourseAnalyticsQuery aggregates analytics across all activities in a course.
func (r *Repository) CourseAnalyticsQuery(ctx context.Context, cursoID uuid.UUID, docenteID uuid.UUID) (CourseAnalytics, error) {
	out := CourseAnalytics{CursoID: cursoID}

	rows, err := r.db.Query(ctx,
		`SELECT id, tipo FROM actividades WHERE curso_id = $1 ORDER BY created_at`,
		cursoID,
	)
	if err != nil {
		return out, fmt.Errorf("list actividades: %w", err)
	}
	defer rows.Close()

	type actRow struct {
		id   uuid.UUID
		tipo string
	}
	var acts []actRow
	for rows.Next() {
		var a actRow
		if err := rows.Scan(&a.id, &a.tipo); err != nil {
			return out, err
		}
		acts = append(acts, a)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	var totalFlagged int
	for _, a := range acts {
		aa, err := r.ActivityAnalyticsQuery(ctx, a.id, a.tipo)
		if err != nil {
			return out, err
		}
		out.Activities = append(out.Activities, aa)
		out.TotalSubmissions += aa.SubmissionCount
		totalFlagged += aa.FlaggedCount
	}

	// Course-level avg grade across all graded submissions
	var avg *float64
	_ = r.db.QueryRow(ctx, `
		SELECT AVG(g) FROM (
			SELECT gd.final_grade AS g
			FROM grade_decisions gd
			JOIN exam_sessions es ON es.id = gd.session_id
			JOIN actividades a ON a.id = es.actividad_id
			WHERE a.curso_id = $1
			UNION ALL
			SELECT ss.grade FROM stage_submissions ss
			JOIN investigacion_stages s ON ss.stage_id = s.id
			JOIN actividades a ON s.actividad_id = a.id
			WHERE a.curso_id = $1 AND ss.grade IS NOT NULL
			UNION ALL
			SELECT ga.grade FROM git_analyses ga
			JOIN actividades a ON ga.actividad_id = a.id
			WHERE a.curso_id = $1 AND ga.grade IS NOT NULL
			UNION ALL
			SELECT da.grade FROM defense_appointments da
			JOIN actividades a ON da.actividad_id = a.id
			WHERE a.curso_id = $1 AND da.grade IS NOT NULL
		) grades`, cursoID,
	).Scan(&avg)
	out.AvgGrade = avg

	if out.TotalSubmissions > 0 {
		out.FlagRate = float64(totalFlagged) / float64(out.TotalSubmissions)
	}

	return out, nil
}
