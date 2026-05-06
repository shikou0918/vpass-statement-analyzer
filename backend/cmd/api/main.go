package main

import (
	"log"
	"net/http"
	"os"

	httpadapter "vpass-statement-analyzer/backend/internal/adapter/http"
	"vpass-statement-analyzer/backend/internal/config"
	"vpass-statement-analyzer/backend/internal/infrastructure/database"
	"vpass-statement-analyzer/backend/internal/usecase"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	repos := database.NewRepositories(db)
	tx := database.NewTxManager(db)
	uc := usecase.NewApp(repos, tx)
	router := httpadapter.NewRouter(uc, cfg.AllowedOrigin)

	log.Printf("server listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
