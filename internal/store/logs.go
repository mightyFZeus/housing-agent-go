package store

import (
	"context"

	"github.com/mightyfzeus/housing-agent/internal/models"
	"gorm.io/gorm"
)

type LogStore struct {
	db *gorm.DB
}

func (ds *LogStore) CreateLog(ctx context.Context, log *models.QueryLog) error {
	return ds.db.WithContext(ctx).Create(log).Error
}
