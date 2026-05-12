package app

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"todo/db"
	pkglog "todo/pkg/log"
	"go.uber.org/zap"
)

func safeURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if !strings.HasPrefix(u.Scheme, "http") {
		return ""
	}
	return rawURL
}

type TodoFull struct {
	db.Todo
	Tags []db.TodoTag `json:"tags"`
	Logs []db.TodoLog `json:"logs"`
}

func RefreshFull(id int64) (*TodoFull, error) {
	t, err := db.GetTodo(id)
	if err != nil {
		pkglog.GetLogger(nil).Error("RefreshFull: get todo failed", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("failed to load todo")
	}
	if t == nil {
		return nil, errors.New("not found")
	}
	tags, err := db.GetTags(id)
	if err != nil {
		pkglog.GetLogger(nil).Error("RefreshFull: get tags failed", zap.Int64("id", id), zap.Error(err))
		// tags optional
	}
	logs, err := db.GetLogs(id)
	if err != nil {
		pkglog.GetLogger(nil).Error("RefreshFull: get logs failed", zap.Int64("id", id), zap.Error(err))
	}
	return &TodoFull{*t, tags, logs}, nil
}

func Create(content, threadLink string, scheduleAt int64) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	if content == "" {
		err := errors.New("content required")
		logger.Error("Create: empty content")
		return nil, err
	}
	threadLink = safeURL(threadLink)
	t, err := db.CreateTodo(content, threadLink, scheduleAt)
	if err != nil {
		logger.Error("Create: db create failed", zap.Error(err))
		return nil, errors.New("failed to create todo")
	}
	return RefreshFull(t.ID)
}

func Start(id int64) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	t, err := db.GetTodo(id)
	if err != nil || t == nil {
		logger.Error("Start: todo not found", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("not found")
	}
	if t.Status != "open" {
		err := fmt.Errorf("cannot start: todo is %s", t.Status)
		logger.Error("Start: invalid state", zap.Int64("id", id), zap.String("status", t.Status))
		return nil, err
	}
	if err := db.UpdateStatus(id, "doing", 0, 0); err != nil {
		logger.Error("Start: update status failed", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("failed to update status")
	}
	_, _ = db.AddLog(id, "start", "")
	return RefreshFull(id)
}

func Pause(id int64) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	t, err := db.GetTodo(id)
	if err != nil || t == nil {
		logger.Error("Pause: todo not found", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("not found")
	}
	if t.Status != "doing" {
		err := fmt.Errorf("cannot pause: todo is %s", t.Status)
		logger.Error("Pause: invalid state", zap.Int64("id", id), zap.String("status", t.Status))
		return nil, err
	}
	if err := db.UpdateStatus(id, "open", 0, 0); err != nil {
		logger.Error("Pause: update status failed", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("failed to update status")
	}
	_, _ = db.AddLog(id, "pause", "")
	return RefreshFull(id)
}

func Complete(id int64, note string) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	t, err := db.GetTodo(id)
	if err != nil || t == nil {
		logger.Error("Complete: todo not found", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("not found")
	}
	if t.Status == "completed" {
		err := errors.New("already completed")
		logger.Error("Complete: already completed", zap.Int64("id", id))
		return nil, err
	}
	now := time.Now().Unix()
	if err := db.UpdateStatus(id, "completed", 0, now); err != nil {
		logger.Error("Complete: update status failed", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("failed to update status")
	}
	_, _ = db.AddLog(id, "complete", note)
	return RefreshFull(id)
}

func Reopen(id int64) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	t, err := db.GetTodo(id)
	if err != nil || t == nil {
		logger.Error("Reopen: todo not found", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("not found")
	}
	if t.Status != "completed" {
		err := errors.New("can only reopen completed todos")
		logger.Error("Reopen: invalid state", zap.Int64("id", id), zap.String("status", t.Status))
		return nil, err
	}
	if err := db.UpdateStatus(id, "open", 0, 0); err != nil {
		logger.Error("Reopen: update status failed", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("failed to update status")
	}
	_, _ = db.AddLog(id, "reopen", "")
	return RefreshFull(id)
}

func AddNote(id int64, note string) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	if note == "" {
		err := errors.New("note cannot be empty")
		logger.Error("AddNote: empty note", zap.Int64("id", id))
		return nil, err
	}
	t, err := db.GetTodo(id)
	if err != nil || t == nil {
		logger.Error("AddNote: todo not found", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("not found")
	}
	_, err = db.AddLog(id, "note", note)
	if err != nil {
		logger.Error("AddNote: add log failed", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("failed to add note")
	}
	return RefreshFull(id)
}

func DeleteNote(id, logID int64) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	t, err := db.GetTodo(id)
	if err != nil || t == nil {
		logger.Error("DeleteNote: todo not found", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("not found")
	}
	if err := db.DeleteLog(logID); err != nil {
		logger.Error("DeleteNote: delete log failed", zap.Int64("log_id", logID), zap.Error(err))
		return nil, errors.New("failed to delete note")
	}
	return RefreshFull(id)
}

func SetThreadLink(id int64, link string) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	t, err := db.GetTodo(id)
	if err != nil || t == nil {
		logger.Error("SetThreadLink: todo not found", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("not found")
	}
	link = safeURL(link)
	if err := db.SetThreadLink(id, link); err != nil {
		logger.Error("SetThreadLink: db error", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("failed to set thread link")
	}
	return RefreshFull(id)
}

func AddTagToTodo(id int64, group, tag string) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	t, err := db.GetTodo(id)
	if err != nil || t == nil {
		logger.Error("AddTag: todo not found", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("not found")
	}
	if group == "" || tag == "" {
		err := errors.New("group and tag required")
		logger.Error("AddTag: missing group/tag", zap.Int64("id", id))
		return nil, err
	}
	if err := db.AddTag(id, group, tag); err != nil {
		logger.Error("AddTag: db error", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("failed to add tag")
	}
	return RefreshFull(id)
}

func RemoveTagFromTodo(id, tagID int64) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	t, err := db.GetTodo(id)
	if err != nil || t == nil {
		logger.Error("RemoveTag: todo not found", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("not found")
	}
	if err := db.RemoveTag(tagID); err != nil {
		logger.Error("RemoveTag: db error", zap.Int64("tag_id", tagID), zap.Error(err))
		return nil, errors.New("failed to remove tag")
	}
	return RefreshFull(id)
}

func SetSchedule(id int64, scheduleAt int64) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	t, err := db.GetTodo(id)
	if err != nil || t == nil {
		logger.Error("SetSchedule: todo not found", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("not found")
	}
	if t.Status != "open" {
		err := fmt.Errorf("schedule only allowed for open todos, current: %s", t.Status)
		logger.Error("SetSchedule: invalid state", zap.Int64("id", id), zap.String("status", t.Status))
		return nil, err
	}
	if err := db.UpdateStatus(id, "open", scheduleAt, 0); err != nil {
		logger.Error("SetSchedule: update failed", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("failed to update schedule")
	}
	if scheduleAt > 0 {
		_, _ = db.AddLog(id, "schedule", fmt.Sprintf("set to %s", time.Unix(scheduleAt, 0).Format("2006-01-02 15:04")))
	} else {
		_, _ = db.AddLog(id, "schedule", "cleared")
	}
	return RefreshFull(id)
}

func Later(id int64, scheduleAt int64) (*TodoFull, error) {
	logger := pkglog.GetLogger(nil)
	if scheduleAt == 0 {
		err := errors.New("later requires a target time")
		logger.Error("Later: missing target time", zap.Int64("id", id))
		return nil, err
	}
	t, err := db.GetTodo(id)
	if err != nil || t == nil {
		logger.Error("Later: todo not found", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("not found")
	}
	if t.Status != "doing" {
		err := fmt.Errorf("later only allowed for doing todos, current: %s", t.Status)
		logger.Error("Later: invalid state", zap.Int64("id", id), zap.String("status", t.Status))
		return nil, err
	}
	if err := db.UpdateStatus(id, "open", scheduleAt, 0); err != nil {
		logger.Error("Later: update failed", zap.Int64("id", id), zap.Error(err))
		return nil, errors.New("failed to update schedule")
	}
	_, _ = db.AddLog(id, "later", fmt.Sprintf("postponed to %s", time.Unix(scheduleAt, 0).Format("2006-01-02 15:04")))
	return RefreshFull(id)
}

func Delete(id int64) error {
	err := db.DeleteTodo(id)
	if err != nil {
		pkglog.GetLogger(nil).Error("Delete: failed", zap.Int64("id", id), zap.Error(err))
		return errors.New("failed to delete todo")
	}
	return nil
}
