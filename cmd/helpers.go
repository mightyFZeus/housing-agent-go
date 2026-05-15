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
	apiKey := env.GetString("OPENROUTER_API_KEY", "")
	if apiKey == "" {
		log.Printf("OPENROUTER_API_KEY environment variable is not set")
		return nil, ""
	}

	baseURL := env.GetString("OPENROUTER_API_BASE_URL", "https://openrouter.ai/api/v1")

	model := env.GetString("OPENROUTER_API_MODEL", "text-embedding-3-small")

	client := openai.NewClient(
		option.WithBaseURL(baseURL),
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
