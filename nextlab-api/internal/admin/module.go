package admin

import (
	"github.com/Elder2003/nextlab-api/internal/rag"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

type Module struct {
	repo    *Repository
	curRepo *rag.CurricularRepository
	mc      *minio.Client
	bucket  string
	appEnv  string
}

func NewModule(db *pgxpool.Pool, mc *minio.Client, bucket, appEnv string) *Module {
	return &Module{
		repo:    NewRepository(db),
		curRepo: rag.NewCurricularRepository(db),
		mc:      mc,
		bucket:  bucket,
		appEnv:  appEnv,
	}
}

func (m *Module) RegisterRoutes(app *fiber.App, jwtMW fiber.Handler, roleGuard func(...string) fiber.Handler) {
	admin := app.Group("/admin", jwtMW, roleGuard("admin"))

	// US1: Docente account management
	admin.Post("/docentes", CreateDocenteHandler(m.repo, m.appEnv))
	admin.Get("/docentes", ListDocentesHandler(m.repo))
	admin.Get("/docentes/:id", GetDocenteHandler(m.repo))
	admin.Put("/docentes/:id", UpdateDocenteHandler(m.repo))
	admin.Delete("/docentes/:id", DeleteDocenteHandler(m.repo))

	// US2: Format template library (admin endpoints)
	admin.Post("/format-templates", CreateFormatTemplateHandler(m.repo))
	admin.Get("/format-templates", ListFormatTemplatesHandler(m.repo))
	admin.Get("/format-templates/:id", GetFormatTemplateHandler(m.repo))
	admin.Put("/format-templates/:id", UpdateFormatTemplateHandler(m.repo))
	admin.Delete("/format-templates/:id", DeleteFormatTemplateHandler(m.repo))

	// US2: Format template apply (docente endpoints)
	docente := app.Group("/actividades", jwtMW, roleGuard("docente"))
	docente.Post("/:aid/format-template", ApplyTemplateHandler(m.repo))
	docente.Get("/:aid/format-template", GetAppliedTemplateHandler(m.repo))

	// US3: Curriculum document management
	admin.Get("/curriculum/documents", ListCurriculumDocsHandler(m.curRepo))
	admin.Delete("/curriculum/documents/:id", DeleteCurriculumDocHandler(m.curRepo, m.mc, m.bucket))

	// US4: Interdisciplinary activity configuration
	admin.Post("/actividades/:aid/interdisciplinary", SetScopesHandler(m.repo))
	admin.Delete("/actividades/:aid/interdisciplinary", DeleteScopesHandler(m.repo))

	shared := app.Group("/actividades", jwtMW, roleGuard("admin", "docente"))
	shared.Get("/:aid/interdisciplinary", GetScopesHandler(m.repo))
}
