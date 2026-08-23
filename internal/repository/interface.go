package repository

import (
	"context"

	"pwni-file-sync/internal/entity"
)

type BinaryFileRepository interface {
	GetByID(ctx context.Context, binID int64) (*entity.BinaryFile, error)
	GetByFileName(ctx context.Context, fileName string) (*entity.BinaryFile, error)
	GetByReferensiID(ctx context.Context, referensiID string, module string) ([]entity.BinaryFile, error)
	GetPendingFiles(ctx context.Context, lastBinID int64, limit int) ([]entity.BinaryFile, error)
	Insert(ctx context.Context, file *entity.BinaryFile) (int64, error)
	UpdatePathAndStatus(ctx context.Context, binID int64, path string, checksum string, size int64, flag string) error
	ExistsByPath(ctx context.Context, path string) (bool, error)
}

type LogRepository interface {
	GetLastBinID(ctx context.Context, server string) (int64, error)
	Insert(ctx context.Context, log entity.LogFileRsync) error
}
