package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/mightyfzeus/housing-agent/internal/store"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type application struct {
	logger     *zap.SugaredLogger
	config     config
	middleWare middleWareConfig
	store      store.Storage
}

type config struct {
	addr   string
	apiUrl string
	db     dbConfig
	env    string
}

type dbConfig struct {
	dbAddr       string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}
type middleWareConfig struct {
	userLocks    sync.Map
	rateLimiters map[string]*rate.Limiter
	rlMu         sync.Mutex
}

func (app *application) mount() http.Handler {

	origins := []string{
		"https://housing-agent-fe.netlify.app",
		"http://localhost:5173",
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Post("/search", app.SearchHandler)
	r.Get("/health", app.HealthHandler)

	return r
}

func (app *application) run(mux http.Handler) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: 70 * time.Second,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  time.Minute,
	}

	return srv.ListenAndServe()
}
