package academic

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Elder2003/nextlab-api/internal/config"
)

// RegisterRoutes wires all M3 academic management endpoints.
// authMW and roleGuard come from the auth package (passed from main to avoid circular imports).
func RegisterRoutes(
	app *fiber.App,
	db *pgxpool.Pool,
	_ *redis.Client,
	_ *config.Config,
	authMW fiber.Handler,
	roleGuard func(roles ...string) fiber.Handler,
) {
	// ─── Admin catalog + student profiles ────────────────────────────────────
	admin := app.Group("/admin", authMW, roleGuard("admin"))

	admin.Get("/carreras", ListCarrerasHandler(db))
	admin.Post("/carreras", CreateCarreraHandler(db))
	admin.Put("/carreras/:id", UpdateCarreraHandler(db))
	admin.Delete("/carreras/:id", DeleteCarreraHandler(db))

	admin.Get("/materias", ListMateriasHandler(db))
	admin.Post("/materias", CreateMateriaHandler(db))
	admin.Put("/materias/:id", UpdateMateriaHandler(db))
	admin.Delete("/materias/:id", DeleteMateriaHandler(db))

	admin.Get("/student-profiles", ListStudentProfilesHandler(db))
	admin.Post("/student-profiles", CreateStudentProfileHandler(db))
	admin.Put("/student-profiles/:id", UpdateStudentProfileHandler(db))

	// ─── Docente — cursos ─────────────────────────────────────────────────────
	cursos := app.Group("/cursos", authMW, roleGuard("docente"))

	cursos.Post("", CreateCursoHandler(db))
	cursos.Get("", ListCursosHandler(db))
	cursos.Get("/:id", GetCursoHandler(db))
	cursos.Put("/:id", UpdateCursoHandler(db))
	cursos.Delete("/:id", DeleteCursoHandler(db))
	cursos.Get("/:id/students", GetCursoRosterHandler(db))

	// ─── Docente — grupos ─────────────────────────────────────────────────────
	cursos.Post("/:id/grupos", CreateGrupoHandler(db))
	cursos.Get("/:id/grupos", ListGruposHandler(db))
	cursos.Put("/:id/grupos/:gid", UpdateGrupoHandler(db))
	cursos.Delete("/:id/grupos/:gid", DeleteGrupoHandler(db))
	cursos.Post("/:id/grupos/:gid/students", AssignStudentHandler(db))
	cursos.Delete("/:id/grupos/:gid/students/:uid", RemoveStudentHandler(db))

	// ─── Docente — actividades ────────────────────────────────────────────────
	cursos.Post("/:id/actividades", CreateActividadHandler(db))
	cursos.Get("/:id/actividades", ListActividadesHandler(db))
	cursos.Get("/:id/actividades/:aid", GetActividadHandler(db))
	cursos.Put("/:id/actividades/:aid", UpdateActividadHandler(db))
	cursos.Delete("/:id/actividades/:aid", DeleteActividadHandler(db))

	// ─── Docente — co-teaching grants ─────────────────────────────────────────
	docenteMW := []fiber.Handler{authMW, roleGuard("docente")}
	app.Post("/actividades/:aid/grants", append(docenteMW, CreateGrantHandler(db))...)
	app.Delete("/actividades/:aid/grants/:uid", append(docenteMW, RevokeGrantHandler(db))...)

	// ─── Co-docente — activity view ───────────────────────────────────────────
	grants := app.Group("/grants", authMW, roleGuard("co_docente"))
	grants.Get("/actividades/:aid", CodocenteGetActividadHandler(db))

	// ─── Student — course and activity view ───────────────────────────────────
	me := app.Group("/me", authMW, roleGuard("estudiante"))
	me.Get("/cursos", StudentListCursosHandler(db))
	me.Get("/cursos/:id/actividades", StudentListActividadesHandler(db))
}
