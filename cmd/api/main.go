package main

import (
	"github.com/annguyen0511/social/internal/db"
	"github.com/annguyen0511/social/internal/env"
	"github.com/annguyen0511/social/internal/store"
	"go.uber.org/zap"
)

const version = "0.0.1"

//	@title			Social Network API
//	@version		0.0.1
//	@description	API for building social network.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						Authorization
// @description				Enter your JWT token here to access protected, Example: "Bearer {token}"
func main() {
	cfg := config{
		addr:   env.GetString("ADDR", ":8080"),
		apiURL: env.GetString("EXTERNAL_URL", "http://localhost:8080"),
		dbConfig: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://localhost:5432/socialnetwork?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 10),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
		env: env.GetString("ENV", "development"),
	}

	//logger
	logger := zap.Must(zap.NewProduction()).Sugar()

	defer logger.Sync()

	// database
	db, err := db.New(cfg.dbConfig.addr, cfg.dbConfig.maxOpenConns, cfg.dbConfig.maxIdleConns, cfg.dbConfig.maxIdleTime)

	if err != nil {
		logger.Panicw("failed to connect database", "error", err)
	}

	defer db.Close()

	logger.Infoln("database connection pool establised")
	store := store.NewStorage(db)

	app := &application{
		config: cfg,
		store:  store,
		logger: logger,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))
}
