package format

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

// NewFormatService constructs a FormatService with all dependencies wired.
func NewService(db *pgxpool.Pool, minioClient *minio.Client, bucket string) *FormatService {
	repo := NewRepository(db)
	return NewFormatService(repo, minioClient, bucket)
}
