package app

import (
	"time"
	"todo/db"
	"todo/config"
	pkglog "todo/pkg/log"
	"go.uber.org/zap"
)

func StartWorker() {
	cfg := config.Get()
	interval := time.Duration(cfg.WorkerInterval) * time.Second
	ticker := time.NewTicker(interval)
	logger := pkglog.GetLogger(nil)
	go func() {
		for range ticker.C {
			now := time.Now().Unix()
			ids, err := db.GetDueTodos(now, 20)
			if err != nil {
				logger.Error("worker scan error", zap.Error(err))
				continue
			}
			for _, id := range ids {
				if _, err := Start(id); err != nil {
					logger.Error("auto-start failed", zap.Int64("id", id), zap.Error(err))
				} else {
					logger.Info("auto-started todo", zap.Int64("id", id))
				}
			}
		}
	}()
}
