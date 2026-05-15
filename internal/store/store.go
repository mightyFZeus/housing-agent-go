package store

import (
	"context"

	"github.com/mightyfzeus/housing-agent/internal/models"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type Storage struct {
	Document interface {
		CreateDocment(ctx context.Context, doc *models.Document) error
		Count(ctx context.Context) (int64, error)
		Get(ctx context.Context, qVec pgvector.Vector) ([]*models.Document, error)
	}
}

func NewStorage(db *gorm.DB) Storage {
	return Storage{
		Document: &DocumentStore{db: db},
	}
}
