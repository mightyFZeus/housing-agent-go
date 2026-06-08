package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/mightyfzeus/housing-agent/internal/data"
	"github.com/mightyfzeus/housing-agent/internal/env"
	"github.com/mightyfzeus/housing-agent/internal/models"
	"github.com/openai/openai-go/v2"
	"github.com/pgvector/pgvector-go"
)

func (app *application) EmbedDocuments() {
	ctx := context.Background()

	client, model := app.openAiClient()
	if client == nil {
		app.logger.Error("Error creating OpenAI client")

		return
	}

	count, err := app.store.Document.Count(ctx)
	if err != nil {
		log.Printf("Error counting documents: %v", err)
		return
	}
	if count > 0 {
		return
	}

	const workerCount = 10
	jobs := make(chan models.RawChunk)

	var expectedDim int64
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunk := range jobs {
				resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
					Model: model,
					Input: openai.EmbeddingNewParamsInputUnion{
						OfString: openai.String(chunk.Text),
					},
				})
				if err != nil {
					log.Printf("Error creating embedding for %s: %v", chunk.ID, err)
					continue
				}
				if resp == nil || len(resp.Data) == 0 {
					log.Printf("No embedding returned for %s", chunk.ID)
					continue
				}
				embedding := resp.Data[0].Embedding
				dim := int64(len(embedding))
				if atomic.LoadInt64(&expectedDim) == 0 {
					atomic.CompareAndSwapInt64(&expectedDim, 0, dim)
				} else if atomic.LoadInt64(&expectedDim) != dim {
					log.Printf("Error creating document for %s: embedding dimension mismatch expected %d, got %d", chunk.ID, atomic.LoadInt64(&expectedDim), dim)
					continue
				}
				vec := pgvector.NewVector(ToFloat32Vector(embedding))
				doc := models.Document{
					ID:        uuid.New(),
					Content:   chunk.Text,
					Embedding: vec,
				}
				if err := app.store.Document.CreateDocment(ctx, &doc); err != nil {
					log.Printf("Error creating document for %s: %v", chunk.ID, err)
					continue
				}
			}
		}()
	}

	for _, chunk := range data.Chunks {
		jobs <- chunk
	}
	close(jobs)

	wg.Wait()
}

func (app *application) SearchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		app.internalServerError(w, r, errors.New("streaming unsupported"))
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		app.badRequestResponse(w, r, errors.New("query is required"))
		return
	}
	if err := app.validateQuery(query); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if app.isInjectionAttempt(query) {
		app.badRequestResponse(w, r, errors.New("query not allowed"))
		return
	}
	query = app.sanitizeContext(query)

	client, model := app.openAiClient()
	if client == nil {
		app.internalServerError(w, r, errors.New("openai client is nil"))
		return
	}

	// embeddings
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: model,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(query),
		},
	})
	if err != nil || len(resp.Data) == 0 {
		app.internalServerError(w, r, errors.New("embedding failed"))
		return
	}

	qVec := pgvector.NewVector(ToFloat32Vector(resp.Data[0].Embedding))

	doc, err := app.store.Document.Get(ctx, qVec)
	if err != nil || len(doc) == 0 {
		fmt.Fprintf(w, "data: I don't know\n\n")
		flusher.Flush()
		return
	}

	chatModel := env.GetString("OPEN_AI_CHAT_MODEL", "")
	if chatModel == "" {
		app.internalServerError(w, r, errors.New("missing chat model"))
		return
	}

	// chunk context before seding to llm model
	var context strings.Builder
	for _, d := range doc {
		context.WriteString("- ")
		context.WriteString(d.Content)
		context.WriteString("\n")
	}

	// stream chat
	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model: chatModel,

		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompt),
			openai.UserMessage(fmt.Sprintf(`
Context:
%s
Question:
%s
`, context.String(), query)),
		},
	})

	var full strings.Builder

	for stream.Next() {
		event := stream.Current()

		for _, choice := range event.Choices {
			chunk := choice.Delta.Content
			if chunk == "" {
				continue
			}

			full.WriteString(chunk)

			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
	}

	if err := stream.Err(); err != nil {
		fmt.Fprintf(w, "data: [ERROR] %v\n\n", err)
		flusher.Flush()
		return
	}

	// optional end marker
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}
