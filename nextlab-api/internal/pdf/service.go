package pdf

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// ─── Activity Report ─────────────────────────────────────────────────────────

type submissionRow struct {
	StudentName string
	Status      string
	Grade       *float64
	SubmittedAt time.Time
	IsLate      bool
}

type actividadMeta struct {
	NombreActividad string
	NombreCurso     string
	TipoActividad   string
}

func (s *Service) GenerateActivityReport(ctx context.Context, actividadID uuid.UUID) ([]byte, error) {
	var meta actividadMeta
	if err := s.db.QueryRow(ctx, `
		SELECT a.nombre, c.nombre, a.tipo
		FROM actividades a
		JOIN cursos c ON c.id = a.curso_id
		WHERE a.id = $1
	`, actividadID).Scan(&meta.NombreActividad, &meta.NombreCurso, &meta.TipoActividad); err != nil {
		return nil, fmt.Errorf("fetch actividad: %w", err)
	}

	rows, err := s.db.Query(ctx, `
		SELECT
			COALESCE(u.nombre || ' ' || u.apellido, u.email) AS student_name,
			sub.status,
			sub.grade,
			sub.created_at,
			sub.is_late
		FROM submissions sub
		JOIN users u ON u.id = sub.student_id
		WHERE sub.actividad_id = $1
		ORDER BY u.apellido, u.nombre
	`, actividadID)
	if err != nil {
		return nil, fmt.Errorf("fetch submissions: %w", err)
	}
	defer rows.Close()

	var submissions []submissionRow
	var totalGrade float64
	var gradedCount int
	for rows.Next() {
		var sr submissionRow
		if err := rows.Scan(&sr.StudentName, &sr.Status, &sr.Grade, &sr.SubmittedAt, &sr.IsLate); err != nil {
			return nil, err
		}
		if sr.Grade != nil {
			totalGrade += *sr.Grade
			gradedCount++
		}
		submissions = append(submissions, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var avgGrade float64
	if gradedCount > 0 {
		avgGrade = totalGrade / float64(gradedCount)
	}

	cfg := config.NewBuilder().WithPageSize("A4").Build()
	mrt := maroto.New(cfg)

	mrt.AddRows(headerRow("Reporte de Actividad"))
	mrt.AddRows(infoRow("Curso", meta.NombreCurso))
	mrt.AddRows(infoRow("Actividad", meta.NombreActividad))
	mrt.AddRows(infoRow("Tipo", meta.TipoActividad))
	mrt.AddRows(infoRow("Generado", time.Now().Format("2006-01-02 15:04")))
	mrt.AddRows(infoRow("Total entregas", fmt.Sprintf("%d", len(submissions))))
	mrt.AddRows(infoRow("Calificadas", fmt.Sprintf("%d", gradedCount)))
	if gradedCount > 0 {
		mrt.AddRows(infoRow("Promedio", fmt.Sprintf("%.1f", avgGrade)))
	}
	mrt.AddRows(spacerRow())

	// Table header
	mrt.AddRows(tableHeaderRow([]string{"Estudiante", "Estado", "Nota", "Entrega", "Tardía"}))
	for _, sr := range submissions {
		grade := "-"
		if sr.Grade != nil {
			grade = fmt.Sprintf("%.1f", *sr.Grade)
		}
		late := "No"
		if sr.IsLate {
			late = "Sí"
		}
		mrt.AddRows(tableDataRow([]string{
			sr.StudentName,
			sr.Status,
			grade,
			sr.SubmittedAt.Format("2006-01-02"),
			late,
		}))
	}
	if len(submissions) == 0 {
		mrt.AddRows(tableDataRow([]string{"Sin entregas", "", "", "", ""}))
	}

	doc, err := mrt.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}
	return doc.GetBytes(), nil
}

// ─── Detection Report ─────────────────────────────────────────────────────────

func (s *Service) GenerateDetectionReport(ctx context.Context, submissionID uuid.UUID) ([]byte, error) {
	type detectionData struct {
		StudentName    string
		ActividadName  string
		Score          int
		FlaggedCats    []string
		TemporalJSON   []byte
		LLMAdvisory    string
		LLMStatus      string
		SubmittedAt    time.Time
	}

	var d detectionData
	var flaggedRaw []byte
	if err := s.db.QueryRow(ctx, `
		SELECT
			COALESCE(u.nombre || ' ' || u.apellido, u.email),
			a.nombre,
			ar.score,
			ar.flagged_categories,
			ar.temporal_metrics,
			COALESCE(lr.advisory_text, ''),
			COALESCE(lr.llm_status, 'unavailable'),
			sub.created_at
		FROM submissions sub
		JOIN users u ON u.id = sub.student_id
		JOIN actividades a ON a.id = sub.actividad_id
		JOIN exam_sessions es ON es.actividad_id = sub.actividad_id AND es.student_id = sub.student_id
		JOIN analysis_results ar ON ar.session_id = es.id AND ar.analysis_type = 'heuristic'
		LEFT JOIN analysis_results lr ON lr.session_id = es.id AND lr.analysis_type = 'llm'
		WHERE sub.id = $1
		LIMIT 1
	`, submissionID).Scan(
		&d.StudentName, &d.ActividadName, &d.Score,
		&flaggedRaw, &d.TemporalJSON,
		&d.LLMAdvisory, &d.LLMStatus, &d.SubmittedAt,
	); err != nil {
		return nil, fmt.Errorf("fetch detection data: %w", err)
	}

	if len(flaggedRaw) > 0 {
		_ = json.Unmarshal(flaggedRaw, &d.FlaggedCats)
	}

	cfg := config.NewBuilder().WithPageSize("A4").Build()
	mrt := maroto.New(cfg)

	mrt.AddRows(headerRow("Reporte de Detección"))
	mrt.AddRows(infoRow("Estudiante", d.StudentName))
	mrt.AddRows(infoRow("Actividad", d.ActividadName))
	mrt.AddRows(infoRow("Fecha entrega", d.SubmittedAt.Format("2006-01-02 15:04")))
	mrt.AddRows(infoRow("Score heurístico", fmt.Sprintf("%d/100", d.Score)))

	flaggedStr := "Ninguna"
	if len(d.FlaggedCats) > 0 {
		flaggedStr = ""
		for i, f := range d.FlaggedCats {
			if i > 0 {
				flaggedStr += ", "
			}
			flaggedStr += f
		}
	}
	mrt.AddRows(infoRow("Categorías flaggeadas", flaggedStr))
	mrt.AddRows(spacerRow())

	if len(d.TemporalJSON) > 0 {
		var tm map[string]any
		if json.Unmarshal(d.TemporalJSON, &tm) == nil {
			mrt.AddRows(subHeaderRow("Métricas Temporales"))
			for k, v := range tm {
				mrt.AddRows(infoRow(k, fmt.Sprintf("%v", v)))
			}
			mrt.AddRows(spacerRow())
		}
	}

	advisory := d.LLMAdvisory
	if advisory == "" || d.LLMStatus != "success" {
		advisory = "[Advisory no disponible]"
	}
	mrt.AddRows(subHeaderRow("Advisory LLM"))
	mrt.AddRows(longTextRow(advisory))

	doc, err := mrt.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}
	return doc.GetBytes(), nil
}

// ─── Appeal Audit Report ──────────────────────────────────────────────────────

func (s *Service) GenerateAppealAuditReport(ctx context.Context, appealID uuid.UUID) ([]byte, error) {
	type appealMeta struct {
		Type      string
		Status    string
		Reason    string
		CreatedAt time.Time
		FiledBy   string
	}
	var am appealMeta
	if err := s.db.QueryRow(ctx, `
		SELECT
			ap.type, ap.status, ap.reason, ap.created_at,
			COALESCE(u.nombre || ' ' || u.apellido, u.email)
		FROM appeals ap
		JOIN users u ON u.id = ap.filed_by
		WHERE ap.id = $1
	`, appealID).Scan(&am.Type, &am.Status, &am.Reason, &am.CreatedAt, &am.FiledBy); err != nil {
		return nil, fmt.Errorf("fetch appeal: %w", err)
	}

	auditRows, err := s.db.Query(ctx, `
		SELECT sequence, event_type, occurred_at, event_hash, prev_hash
		FROM appeal_audit_events
		WHERE appeal_id = $1
		ORDER BY sequence ASC
	`, appealID)
	if err != nil {
		return nil, fmt.Errorf("fetch audit events: %w", err)
	}
	defer auditRows.Close()

	type auditEntry struct {
		Seq       int
		EventType string
		OccurredAt time.Time
		Hash      string
		PrevHash  *string
	}
	var entries []auditEntry
	for auditRows.Next() {
		var e auditEntry
		if err := auditRows.Scan(&e.Seq, &e.EventType, &e.OccurredAt, &e.Hash, &e.PrevHash); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if err := auditRows.Err(); err != nil {
		return nil, err
	}

	cfg := config.NewBuilder().WithPageSize("A4").Build()
	mrt := maroto.New(cfg)

	mrt.AddRows(headerRow("Reporte de Auditoría de Apelación"))
	mrt.AddRows(infoRow("Tipo", am.Type))
	mrt.AddRows(infoRow("Estado", am.Status))
	mrt.AddRows(infoRow("Presentado por", am.FiledBy))
	mrt.AddRows(infoRow("Fecha", am.CreatedAt.Format("2006-01-02 15:04")))
	mrt.AddRows(infoRow("Motivo", am.Reason))
	mrt.AddRows(spacerRow())

	mrt.AddRows(subHeaderRow("Cadena de Auditoría"))
	mrt.AddRows(tableHeaderRow([]string{"Seq", "Evento", "Fecha", "Hash"}))
	for _, e := range entries {
		hashShort := e.Hash
		if len(hashShort) > 16 {
			hashShort = hashShort[:16] + "..."
		}
		mrt.AddRows(tableDataRow([]string{
			fmt.Sprintf("%d", e.Seq),
			e.EventType,
			e.OccurredAt.Format("2006-01-02 15:04:05"),
			hashShort,
		}))
	}
	if len(entries) == 0 {
		mrt.AddRows(tableDataRow([]string{"Sin eventos de auditoría", "", "", ""}))
	}

	doc, err := mrt.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}
	return doc.GetBytes(), nil
}

// ─── Maroto helpers ───────────────────────────────────────────────────────────

func headerRow(title string) core.Row {
	return row.New(12).Add(
		col.New(12).Add(text.New(title, props.Text{Size: 16, Style: "bold", Align: "center"})),
	)
}

func subHeaderRow(title string) core.Row {
	return row.New(8).Add(
		col.New(12).Add(text.New(title, props.Text{Size: 12, Style: "bold"})),
	)
}

func infoRow(label, value string) core.Row {
	return row.New(6).Add(
		col.New(4).Add(text.New(label+":", props.Text{Size: 9, Style: "bold"})),
		col.New(8).Add(text.New(value, props.Text{Size: 9})),
	)
}

func spacerRow() core.Row {
	return row.New(4).Add(col.New(12))
}

func longTextRow(content string) core.Row {
	if len(content) > 300 {
		content = content[:300] + "..."
	}
	return row.New(20).Add(
		col.New(12).Add(text.New(content, props.Text{Size: 9})),
	)
}

func tableHeaderRow(cols []string) core.Row {
	cells := make([]core.Col, len(cols))
	w := 12 / len(cols)
	for i, c := range cols {
		cells[i] = col.New(w).Add(text.New(c, props.Text{Size: 9, Style: "bold"}))
	}
	return row.New(7).Add(cells...)
}

func tableDataRow(cols []string) core.Row {
	cells := make([]core.Col, len(cols))
	w := 12 / len(cols)
	for i, c := range cols {
		cells[i] = col.New(w).Add(text.New(c, props.Text{Size: 8}))
	}
	return row.New(6).Add(cells...)
}
