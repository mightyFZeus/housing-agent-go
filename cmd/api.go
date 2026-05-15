package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// secret := env.GetString("SECRET_KEY", "jMdftY0vIiVLTXChDeMYsMo62Jk6XmUnquEfuslkD0xapZo6HWRtq2scWZlyY1cZck4wa5PNQXSnGNdTJs67hw=")

	// Public

	r.Get("/search", app.SearchHandler)
	r.Get("/health", app.HealthHandler)

	return r
}

func (app *application) run(mux http.Handler) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: time.Second * 10,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	return srv.ListenAndServe()
}
