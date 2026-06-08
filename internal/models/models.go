package models

import (
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type Document struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Content    string
	Embedding  pgvector.Vector `gorm:"type:vector"`
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
	Question       string
	RetrievedChunk string
	Distance       float64
	Answer         string
}
