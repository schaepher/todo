package main

import (
	"fmt"
	"net/http"
	"os"
	"todo/app"
	"todo/config"
	"todo/db"
	pkglog "todo/pkg/log"

	"go.uber.org/zap"
)

func main() {
	pkglog.InitLogger()
	defer pkglog.Sync()

	logger := pkglog.GetLogger(nil)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	if err := db.Init(cfg.DBPath()); err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "serve" {
			startServer(cfg, logger)
		} else {
			app.RunCLI(os.Args[1:])
		}
	} else {
		app.RunCLI([]string{})
	}
}

func startServer(cfg *config.Config, logger *zap.Logger) {
	mux := http.NewServeMux()
	app.SetupAPIRoutes(mux)
	app.SetupUIRoutes(mux)
	app.StartWorker()

	handler := app.SecurityHeadersMiddleware(mux)

	addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Info("starting admin server", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, handler); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}
}
