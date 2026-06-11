package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type Document struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Content    string
	Embedding  pgvector.Vector `gorm:"type:vector(1536)"`
	Similarity float64         `gorm:"-"`
	Distance   float64         `gorm:"column:distance"`
}

type RawChunk struct {
	ID      string `json:"id"`
	Section string `json:"section"`
	Title   string `json:"title"`
	Text    string `json:"text"`
	Source  string `json:"source"`
	Page    int    `json:"page"`
}

type QueryLog struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Question       string    `json:"question"`
	RetrievedChunk string    `json:"retrieved_chunk"`
	Distance       float64   `json:"distance"`
	Similarity     string    `json:"similarity"`
	Answer         string    `json:"answer"`
	CreatedAt      time.Time `gorm:"createdAt"`
}
