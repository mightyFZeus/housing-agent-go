package store

import (
	"context"

	"github.com/mightyfzeus/housing-agent/internal/models"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type DocumentStore struct {
	db *gorm.DB
}

func (ds *DocumentStore) CreateDocment(ctx context.Context, doc *models.Document) error {
	return ds.db.WithContext(ctx).Create(doc).Error
}

func (ds *DocumentStore) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := ds.db.WithContext(ctx).Model(&models.Document{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (ds *DocumentStore) Get(ctx context.Context, qVec pgvector.Vector) ([]*models.Document, error) {
	var docs []*models.Document

	err := ds.db.WithContext(ctx).
		Raw(`
			SELECT *
			FROM documents
			ORDER BY embedding <-> ?
			LIMIT 5
		`, qVec).
		Scan(&docs).Error

	if err != nil {
		return nil, err
	}

	return docs, nil
}
