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

// 1. Add rawQuery string to the parameters
func (ds *DocumentStore) Get(ctx context.Context, rawQuery string, qVec pgvector.Vector) ([]*models.Document, error) {
	var docs []*models.Document

	err := ds.db.WithContext(ctx).
		Raw(`
			WITH vector_matches AS (
				SELECT
					id,
					content,
					embedding <=> $1 AS distance,
					ROW_NUMBER() OVER (ORDER BY embedding <=> $1) AS rank
				FROM documents
				ORDER BY embedding <=> $1
				LIMIT 20
			),

			keyword_matches AS (
				SELECT
					id,
					content,
					ts_rank_cd(tsv, plainto_tsquery('english', $2)) AS score,
					ROW_NUMBER() OVER (
						ORDER BY ts_rank_cd(tsv, plainto_tsquery('english', $2)) DESC
					) AS rank
				FROM documents
				WHERE tsv @@ plainto_tsquery('english', $2)
				LIMIT 20
			)

			SELECT
				COALESCE(v.id, k.id) AS id,
				COALESCE(v.content, k.content) AS content,

				COALESCE(v.distance, 1.0) AS distance,

				(
					COALESCE(1.0 / (10 + v.rank), 0.0) +
					COALESCE(1.0 / (10 + k.rank), 0.0)
				) AS rrf_score

			FROM vector_matches v
			FULL OUTER JOIN keyword_matches k ON v.id = k.id

			ORDER BY rrf_score DESC
			LIMIT 5;
		`, qVec, rawQuery).
		Scan(&docs).Error

	if err != nil {
		return nil, err
	}

	return docs, nil
}
