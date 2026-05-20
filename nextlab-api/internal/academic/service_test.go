package academic

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func strPtr(s string) *string { return &s }
func timePtr(t time.Time) *time.Time { return &t }

func TestGuardActivityMutation(t *testing.T) {
	base := &Actividad{
		ID:           uuid.New(),
		CursoID:      uuid.New(),
		Title:        "Examen Parcial",
		Tipo:         "examen",
		Deadline:     time.Now().Add(24 * time.Hour),
		Instructions: "Resolver todos los ejercicios",
		TraceEnabled: true,
	}

	withSubmissions := func() *Actividad {
		a := *base
		a.HasSubmissions = true
		return &a
	}
	noSubmissions := func() *Actividad {
		a := *base
		a.HasSubmissions = false
		return &a
	}

	tests := []struct {
		name     string
		actividad *Actividad
		req      *UpdateActividadRequest
		isDelete bool
		wantErr  error
	}{
		// ── No submissions: guard never fires ──────────────────────────────────
		{
			name:      "no submissions, delete allowed",
			actividad: noSubmissions(),
			req:       nil,
			isDelete:  true,
			wantErr:   nil,
		},
		{
			name:      "no submissions, title change allowed",
			actividad: noSubmissions(),
			req:       &UpdateActividadRequest{Title: strPtr("Nuevo Título")},
			isDelete:  false,
			wantErr:   nil,
		},
		{
			name:      "no submissions, tipo change allowed",
			actividad: noSubmissions(),
			req:       &UpdateActividadRequest{Tipo: strPtr("practica")},
			isDelete:  false,
			wantErr:   nil,
		},
		{
			name:      "no submissions, all fields allowed",
			actividad: noSubmissions(),
			req: &UpdateActividadRequest{
				Title:        strPtr("Nuevo"),
				Tipo:         strPtr("tarea"),
				Deadline:     timePtr(time.Now().Add(48 * time.Hour)),
				Instructions: strPtr("Nuevas instrucciones"),
			},
			isDelete: false,
			wantErr:  nil,
		},
		{
			name:      "no submissions, nil req allowed",
			actividad: noSubmissions(),
			req:       nil,
			isDelete:  false,
			wantErr:   nil,
		},

		// ── Has submissions: delete blocked ────────────────────────────────────
		{
			name:      "has submissions, delete blocked",
			actividad: withSubmissions(),
			req:       nil,
			isDelete:  true,
			wantErr:   ErrActivityHasSubmissions,
		},

		// ── Has submissions: title change blocked ──────────────────────────────
		{
			name:      "has submissions, title change blocked",
			actividad: withSubmissions(),
			req:       &UpdateActividadRequest{Title: strPtr("Otro título")},
			isDelete:  false,
			wantErr:   ErrActivityFieldLocked,
		},

		// ── Has submissions: tipo change blocked ───────────────────────────────
		{
			name:      "has submissions, tipo change blocked",
			actividad: withSubmissions(),
			req:       &UpdateActividadRequest{Tipo: strPtr("tarea")},
			isDelete:  false,
			wantErr:   ErrActivityFieldLocked,
		},

		// ── Has submissions: both title+tipo blocked ───────────────────────────
		{
			name:      "has submissions, title+tipo change blocked",
			actividad: withSubmissions(),
			req:       &UpdateActividadRequest{Title: strPtr("X"), Tipo: strPtr("tarea")},
			isDelete:  false,
			wantErr:   ErrActivityFieldLocked,
		},

		// ── Has submissions: safe fields allowed ───────────────────────────────
		{
			name:     "has submissions, deadline change allowed",
			actividad: withSubmissions(),
			req:       &UpdateActividadRequest{Deadline: timePtr(time.Now().Add(72 * time.Hour))},
			isDelete:  false,
			wantErr:   nil,
		},
		{
			name:      "has submissions, instructions change allowed",
			actividad: withSubmissions(),
			req:       &UpdateActividadRequest{Instructions: strPtr("Instrucciones nuevas")},
			isDelete:  false,
			wantErr:   nil,
		},
		{
			name:     "has submissions, deadline+instructions allowed",
			actividad: withSubmissions(),
			req: &UpdateActividadRequest{
				Deadline:     timePtr(time.Now().Add(48 * time.Hour)),
				Instructions: strPtr("Updated"),
			},
			isDelete: false,
			wantErr:  nil,
		},
		{
			name:      "has submissions, nil req allowed (no-op update)",
			actividad: withSubmissions(),
			req:       nil,
			isDelete:  false,
			wantErr:   nil,
		},
		{
			name:      "has submissions, empty req allowed (no-op update)",
			actividad: withSubmissions(),
			req:       &UpdateActividadRequest{},
			isDelete:  false,
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GuardActivityMutation(tt.actividad, tt.req, tt.isDelete)
			if err != tt.wantErr {
				t.Errorf("GuardActivityMutation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
