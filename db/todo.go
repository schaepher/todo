package db

import (
	"database/sql"
	"time"

	pkglog "todo/pkg/log"
	"go.uber.org/zap"
)

type Todo struct {
	ID           int64  `json:"id"`
	Content      string `json:"content"`
	Status       string `json:"status"`
	CreatedAt    int64  `json:"created_at"`
	ThreadLink   string `json:"thread_link"`
	ScheduledAt  int64  `json:"scheduled_at"`
	CompletedAt  int64  `json:"completed_at"`
}

type TodoLog struct {
	ID        int64  `json:"id"`
	TodoID    int64  `json:"todo_id"`
	Action    string `json:"action"`
	Note      string `json:"note"`
	CreatedAt int64  `json:"created_at"`
}

type TagInput struct {
	GroupName string `json:"group_name"`
	Tag       string `json:"tag"`
}

type TodoTag struct {
	ID        int64  `json:"id"`
	TodoID    int64  `json:"todo_id"`
	GroupName string `json:"group_name"`
	Tag       string `json:"tag"`
}

func CreateTodo(content, threadLink string, scheduledAt int64) (*Todo, error) {
	logger := pkglog.GetLogger(nil)
	now := time.Now().Unix()
	res, err := DB.Exec(`INSERT INTO todos (content, status, created_at, thread_link, scheduled_at) VALUES (?, 'open', ?, ?, ?)`,
		content, now, threadLink, scheduledAt)
	if err != nil {
		logger.Error("create todo failed", zap.String("content", content), zap.Error(err))
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Todo{
		ID:          id,
		Content:     content,
		Status:      "open",
		CreatedAt:   now,
		ThreadLink:  threadLink,
		ScheduledAt: scheduledAt,
	}, nil
}

func GetTodo(id int64) (*Todo, error) {
	row := DB.QueryRow(`SELECT id, content, status, created_at, thread_link, scheduled_at, completed_at FROM todos WHERE id=?`, id)
	t := &Todo{}
	err := row.Scan(&t.ID, &t.Content, &t.Status, &t.CreatedAt, &t.ThreadLink, &t.ScheduledAt, &t.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		pkglog.GetLogger(nil).Error("get todo failed", zap.Int64("id", id), zap.Error(err))
	}
	return t, err
}

func ListTodos(includeCompleted bool) ([]Todo, error) {
	logger := pkglog.GetLogger(nil)
	query := `SELECT id, content, status, created_at, thread_link, scheduled_at, completed_at FROM todos`
	if !includeCompleted {
		query += ` WHERE status != 'completed'`
	}
	query += ` ORDER BY created_at DESC`
	rows, err := DB.Query(query)
	if err != nil {
		logger.Error("list todos query failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()
	var todos []Todo
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.Content, &t.Status, &t.CreatedAt, &t.ThreadLink, &t.ScheduledAt, &t.CompletedAt); err != nil {
			logger.Error("scan todo row failed", zap.Error(err))
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, rows.Err()
}

func UpdateStatus(id int64, status string, scheduledAt, completedAt int64) error {
	_, err := DB.Exec(`UPDATE todos SET status=?, scheduled_at=?, completed_at=? WHERE id=?`, status, scheduledAt, completedAt, id)
	if err != nil {
		pkglog.GetLogger(nil).Error("update status failed", zap.Int64("id", id), zap.String("status", status), zap.Error(err))
	}
	return err
}

func SetThreadLink(id int64, link string) error {
	_, err := DB.Exec(`UPDATE todos SET thread_link=? WHERE id=?`, link, id)
	if err != nil {
		pkglog.GetLogger(nil).Error("set thread link failed", zap.Int64("id", id), zap.Error(err))
	}
	return err
}

func AddLog(todoID int64, action, note string) (int64, error) {
	now := time.Now().Unix()
	res, err := DB.Exec(`INSERT INTO todo_logs (todo_id, action, note, created_at) VALUES (?,?,?,?)`, todoID, action, note, now)
	if err != nil {
		pkglog.GetLogger(nil).Error("add log failed", zap.Int64("todo_id", todoID), zap.String("action", action), zap.Error(err))
		return 0, err
	}
	return res.LastInsertId()
}

func DeleteLog(logID int64) error {
	_, err := DB.Exec(`UPDATE todo_logs SET note='' WHERE id=?`, logID)
	if err != nil {
		pkglog.GetLogger(nil).Error("delete log failed", zap.Int64("log_id", logID), zap.Error(err))
	}
	return err
}

func GetLogs(todoID int64) ([]TodoLog, error) {
	logger := pkglog.GetLogger(nil)
	rows, err := DB.Query(`SELECT id, todo_id, action, note, created_at FROM todo_logs WHERE todo_id=? ORDER BY created_at DESC`, todoID)
	if err != nil {
		logger.Error("get logs query failed", zap.Int64("todo_id", todoID), zap.Error(err))
		return nil, err
	}
	defer rows.Close()
	var logs []TodoLog
	for rows.Next() {
		var l TodoLog
		if err := rows.Scan(&l.ID, &l.TodoID, &l.Action, &l.Note, &l.CreatedAt); err != nil {
			logger.Error("scan log row failed", zap.Error(err))
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func AddTag(todoID int64, groupName, tag string) error {
	_, err := DB.Exec(`INSERT OR IGNORE INTO todo_tags (todo_id, group_name, tag) VALUES (?,?,?)`, todoID, groupName, tag)
	if err != nil {
		pkglog.GetLogger(nil).Error("add tag failed", zap.Int64("todo_id", todoID), zap.String("group", groupName), zap.String("tag", tag), zap.Error(err))
	}
	return err
}

func RemoveTag(tagID int64) error {
	_, err := DB.Exec(`DELETE FROM todo_tags WHERE id=?`, tagID)
	if err != nil {
		pkglog.GetLogger(nil).Error("remove tag failed", zap.Int64("tag_id", tagID), zap.Error(err))
	}
	return err
}

func GetTags(todoID int64) ([]TodoTag, error) {
	logger := pkglog.GetLogger(nil)
	rows, err := DB.Query(`SELECT id, todo_id, group_name, tag FROM todo_tags WHERE todo_id=? ORDER BY group_name, tag`, todoID)
	if err != nil {
		logger.Error("get tags query failed", zap.Int64("todo_id", todoID), zap.Error(err))
		return nil, err
	}
	defer rows.Close()
	var tags []TodoTag
	for rows.Next() {
		var t TodoTag
		if err := rows.Scan(&t.ID, &t.TodoID, &t.GroupName, &t.Tag); err != nil {
			logger.Error("scan tag row failed", zap.Error(err))
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func DeleteTodo(id int64) error {
	_, err := DB.Exec(`DELETE FROM todos WHERE id=?`, id)
	if err != nil {
		pkglog.GetLogger(nil).Error("delete todo failed", zap.Int64("id", id), zap.Error(err))
	}
	return err
}

func GetDueTodos(now int64, limit int) ([]int64, error) {
	logger := pkglog.GetLogger(nil)
	rows, err := DB.Query(`SELECT id FROM todos WHERE status='open' AND scheduled_at > 0 AND scheduled_at <= ? ORDER BY scheduled_at, id LIMIT ?`, now, limit)
	if err != nil {
		logger.Error("get due todos query failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			logger.Error("scan due todo id failed", zap.Error(err))
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func CountOpen() (int, error) {
	var cnt int
	err := DB.QueryRow(`SELECT COUNT(*) FROM todos WHERE status IN ('open','doing')`).Scan(&cnt)
	if err != nil {
		pkglog.GetLogger(nil).Error("count open failed", zap.Error(err))
	}
	return cnt, err
}
