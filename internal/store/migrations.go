package store

import (
	"github.com/mightyfzeus/housing-agent/internal/models"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	const lockID int64 = 712943281

	if err := db.Exec("SELECT pg_advisory_lock(?)", lockID).Error; err != nil {
		return err
	}
	defer db.Exec("SELECT pg_advisory_unlock(?)", lockID)

	// Enable pgvector
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS vector`).Error; err != nil {
		return err
	}

	if err := db.AutoMigrate(&models.Document{}); err != nil {
		return err
	}

	var typmod int64
	err := db.Raw(
		`SELECT a.atttypmod
FROM pg_attribute a
JOIN pg_class c ON c.oid = a.attrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relname = 'documents'
  AND n.nspname = 'public'
  AND a.attname = 'embedding'
  AND a.attnum > 0
  AND NOT a.attisdropped`,
	).Scan(&typmod).Error
	if err != nil {
		return err
	}

	if typmod != -1 {
		if err := db.Exec(`ALTER TABLE "documents" ALTER COLUMN "embedding" TYPE vector USING "embedding"::vector`).Error; err != nil {
			return err
		}
	}

	return nil
}
