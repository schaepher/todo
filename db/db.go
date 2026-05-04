package db

import (
	"database/sql"
	"fmt"
	"time"

	pkglog "todo/pkg/log"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) error {
	logger := pkglog.GetLogger(nil)
	var err error
	DB, err = sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		logger.Error("failed to open database", zap.String("path", dbPath), zap.Error(err))
		return fmt.Errorf("open db: %w", err)
	}
	DB.SetMaxOpenConns(1)
	DB.SetConnMaxLifetime(0)

	if err := migrate(); err != nil {
		logger.Error("migration failed", zap.Error(err))
		return err
	}
	return nil
}

func migrate() error {
	logger := pkglog.GetLogger(nil)
	ddl := `
	CREATE TABLE IF NOT EXISTS todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL,
		status TEXT NOT NULL CHECK(status IN ('open','doing','completed')),
		created_at INTEGER NOT NULL,
		thread_link TEXT DEFAULT '',
		scheduled_at INTEGER DEFAULT 0,
		completed_at INTEGER DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS todo_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		todo_id INTEGER NOT NULL,
		action TEXT NOT NULL,
		note TEXT DEFAULT '',
		created_at INTEGER NOT NULL,
		FOREIGN KEY (todo_id) REFERENCES todos(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS todo_tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		todo_id INTEGER NOT NULL,
		group_name TEXT NOT NULL,
		tag TEXT NOT NULL,
		UNIQUE(todo_id, group_name, tag),
		FOREIGN KEY (todo_id) REFERENCES todos(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_logs_todo ON todo_logs(todo_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_tags_todo ON todo_tags(todo_id);

	PRAGMA journal_mode=WAL;
	PRAGMA foreign_keys=ON;
	`
	_, err := DB.Exec(ddl)
	if err != nil {
		logger.Error("ddl execution failed", zap.Error(err))
	}
	return err
}

func TimeToUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func UnixToTime(ts int64) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0).In(time.Local)
}
