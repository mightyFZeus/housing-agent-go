package main

import (
	"errors"
	"log"
	"strings"

	"github.com/mightyfzeus/housing-agent/internal/env"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

func ToFloat32Vector(v []float64) []float32 {
	out := make([]float32, len(v))
	for i, val := range v {
		out[i] = float32(val)
	}
	return out
}

func (app *application) openAiClient() (*openai.Client, string) {
	apiKey := env.GetString("OPEN_AI_API_KEY", "")
	if apiKey == "" {
		log.Printf("OPEN_AI_API_KEY environment variable is not set")
		return nil, ""
	}

	model := env.GetString("OPEN_AI_EMBEDDING_MODEL", "text-embedding-3-small")

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)
	return &client, model
}

func (app *application) validateQuery(q string) error {
	if len(q) < 2 {
		return errors.New("query too short")
	}
	if len(q) > 500 {
		return errors.New("query too long")
	}
	return nil
}

var blockedPatterns = []string{
	"ignore previous instructions",
	"system prompt",
	"reveal your prompt",
}

func (app *application) isInjectionAttempt(q string) bool {
	q = strings.ToLower(q)
	for _, p := range blockedPatterns {
		if strings.Contains(q, p) {
			return true
		}
	}
	return false
}

func (app *application) sanitizeContext(text string) string {
	bad := []string{
		"ignore previous instructions",
		"reveal system prompt",
	}

	lower := strings.ToLower(text)
	for _, b := range bad {
		if strings.Contains(lower, b) {
			return "[REDACTED CONTENT]"
		}
	}
	return text
}

func classifyDistance(d float64) string {
	switch {
	case d <= 0.10:
		return "near identical (excellent match)"
	case d <= 0.20:
		return "very strong match"
	case d <= 0.30:
		return "good match"
	case d <= 0.40:
		return "weak match"
	case d <= 0.60:
		return "poor match"
	default:
		return "irrelevant"
	}
}
