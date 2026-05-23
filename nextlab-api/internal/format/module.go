package format

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"

	"github.com/Elder2003/nextlab-api/internal/alerts"
)

// NewService constructs a FormatService with all dependencies wired.
func NewService(db *pgxpool.Pool, minioClient *minio.Client, bucket string, alertWriter alerts.AlertWriter) *FormatService {
	repo := NewRepository(db)
	return NewFormatService(repo, minioClient, bucket, alertWriter)
}
