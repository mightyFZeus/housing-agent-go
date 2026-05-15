package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/mightyfzeus/housing-agent/internal/db"
	"github.com/mightyfzeus/housing-agent/internal/env"
	"github.com/mightyfzeus/housing-agent/internal/store"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Could not load .env file, falling back to defaults")
	}

	cfg := config{
		addr:   env.GetString("ADDR", ":8080"),
		apiUrl: env.GetString("API_URL", "localhost:8080"),
		db: dbConfig{
			dbAddr:       env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost:5433/housing_agent_db?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 25),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 25),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
		env: env.GetString("ENV", "development"),
	}

	// logger
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	// db
	gormDB, err := db.New(cfg.db.dbAddr, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	// rdb, err := db.ConnectToRedis(cfg.redisDb.dBAddr, cfg.redisDb.username, cfg.redisDb.password)
	// if err != nil {
	// 	logger.Fatal("failed to redis  to database", zap.Error(err))
	// }
	sqlDB, err := gormDB.DB()
	if err != nil {
		logger.Fatal("error getting sqlDb from gormDB", zap.Error(err))
	}
	defer sqlDB.Close()

	if err := store.AutoMigrate(gormDB); err != nil {
		logger.Fatal("error running migrations", zap.Error(err))
	}
	// if err := db.CreateIndexes(gormDB); err != nil {
	// 	logger.Fatal("error creating indexes", zap.Error(err))
	// }
	defer sqlDB.Close()
	logger.Info("db conncetion pool established")

	// Start the application
	// store
	store := store.NewStorage(gormDB)

	app := &application{
		config: cfg,
		logger: logger,
		middleWare: middleWareConfig{
			rateLimiters: make(map[string]*rate.Limiter),
		},

		store: store,
	}

	mux := app.mount()
	go app.EmbedDocuments()
	logger.Fatal(app.run(mux))
}
