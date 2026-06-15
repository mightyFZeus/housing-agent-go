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
func (ds *DocumentStore) Get(ctx context.Context, rawQuery string, qVec pgvector.Vector) ([]*models.Result, error) {
	var results []*models.Result

	// TODO: Clean up the query for text search: remove quotes and extract key locations
	// to make sure Postgres can locate them even inside messy text.

	err := ds.db.WithContext(ctx).
		Raw(`
			WITH vector_matches AS (
				SELECT
					id,
					content,
					embedding <=> @qVec AS distance,
					ROW_NUMBER() OVER (ORDER BY embedding <=> @qVec) AS rank
				FROM documents
				ORDER BY embedding <=> @qVec
				LIMIT 20
			),

			keyword_matches AS (
				SELECT
					id,
					content,
					ts_rank_cd(tsv, websearch_to_tsquery('english', @rawQuery)) AS score,
					ROW_NUMBER() OVER (
						ORDER BY ts_rank_cd(tsv, websearch_to_tsquery('english', @rawQuery)) DESC
					) AS rank
				FROM documents
				-- FIX 1: websearch_to_tsquery allows flexible OR matching instead of strict AND matching
				WHERE tsv @@ websearch_to_tsquery('english', @rawQuery)
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
			-- FIX 2: Increased from 5 to 8 to ensure multi-hop contexts (Exemptions + Rules) 
			-- can coexist simultaneously in the LLM prompt.
			LIMIT 8; 
		`, map[string]interface{}{
			"qVec":     qVec,
			"rawQuery": rawQuery,
		}).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}
