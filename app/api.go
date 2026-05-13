package app

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"todo/config"
	"todo/db"
	pkglog "todo/pkg/log"
	"go.uber.org/zap"
)

const maxBodySize = 1 << 20 // 1 MB

func SetupAPIRoutes(mux *http.ServeMux) {
	api := http.NewServeMux()
	api.HandleFunc("/admin/api/todos", handleTodos)
	api.HandleFunc("/admin/api/todos/", handleTodoByID)
	api.HandleFunc("/admin/api/token", handleToken)
	mux.Handle("/admin/api/", AuthMiddleware(api))
}

func apiResponse(w http.ResponseWriter, item interface{}, message string) {
	resp := map[string]interface{}{"success": true, "message": message}
	if item != nil {
		resp["item"] = item
	}
	jsonResp(w, resp)
}

func apiError(w http.ResponseWriter, err string, code int) {
	w.WriteHeader(code)
	jsonResp(w, map[string]interface{}{"success": false, "message": err})
}

func handleTodos(w http.ResponseWriter, r *http.Request) {
	logger := pkglog.GetLogger(r.Context())
	switch r.Method {
	case http.MethodGet:
		inc := r.URL.Query().Get("include_completed") == "true"
		todos, err := db.ListTodos(inc)
		if err != nil {
			logger.Error("api list todos failed", zap.Error(err))
			apiError(w, "failed to list todos", 500)
			return
		}
		type listItem struct {
			db.Todo
			Tags []db.TodoTag `json:"tags"`
			Logs []db.TodoLog `json:"logs"`
		}
		items := make([]listItem, len(todos))
		for i, t := range todos {
			tags, _ := db.GetTags(t.ID)
			logs, _ := db.GetLogs(t.ID)
			items[i] = listItem{Todo: t, Tags: tags, Logs: logs}
		}
		jsonResp(w, map[string]interface{}{"success": true, "items": items})
	case http.MethodPost:
		var req struct {
			Content    string        `json:"content"`
			ThreadLink string        `json:"thread_link"`
			ScheduleAt int64         `json:"schedule_at"`
			Tags       []db.TagInput `json:"tags"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("api create: invalid json", zap.Error(err))
			apiError(w, "invalid json", 400)
			return
		}
		tf, err := Create(req.Content, req.ThreadLink, req.ScheduleAt)
		if err != nil {
			logger.Error("api create failed", zap.Error(err))
			apiError(w, err.Error(), 400)
			return
		}
		for _, t := range req.Tags {
			if t.GroupName != "" && t.Tag != "" {
				AddTagToTodo(tf.ID, t.GroupName, t.Tag)
			}
		}
		tf, _ = RefreshFull(tf.ID)
		apiResponse(w, tf, "created")
	default:
		apiError(w, "method not allowed", 405)
	}
}

func handleTodoByID(w http.ResponseWriter, r *http.Request) {
	logger := pkglog.GetLogger(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/admin/api/todos/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		apiError(w, "not found", 404)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		logger.Error("api: invalid id", zap.String("path", path), zap.Error(err))
		apiError(w, "invalid id", 400)
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodDelete:
			if err := Delete(id); err != nil {
				logger.Error("api delete failed", zap.Int64("id", id), zap.Error(err))
				apiError(w, err.Error(), 400)
				return
			}
			apiResponse(w, nil, "deleted")
		default:
			apiError(w, "method not allowed", 405)
		}
		return
	}

	action := parts[1]
	switch action {
	case "note":
		if r.Method == http.MethodPost {
			var req struct {
				Note string `json:"note"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
			json.NewDecoder(r.Body).Decode(&req)
			tf, err := AddNote(id, req.Note)
			if err != nil {
				logger.Error("api add note failed", zap.Int64("id", id), zap.Error(err))
				apiError(w, err.Error(), 400)
				return
			}
			apiResponse(w, tf, "note added")
		} else {
			apiError(w, "method not allowed", 405)
		}
	case "notes":
		if len(parts) >= 3 {
			logID, _ := strconv.ParseInt(parts[2], 10, 64)
			if r.Method == http.MethodDelete {
				tf, err := DeleteNote(id, logID)
				if err != nil {
					logger.Error("api delete note failed", zap.Int64("log_id", logID), zap.Error(err))
					apiError(w, err.Error(), 400)
					return
				}
				apiResponse(w, tf, "note deleted")
			} else {
				apiError(w, "method not allowed", 405)
			}
		} else {
			apiError(w, "not found", 404)
		}
	case "thread":
		if r.Method == http.MethodPost {
			var req struct {
				ThreadLink string `json:"thread_link"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
			json.NewDecoder(r.Body).Decode(&req)
			tf, err := SetThreadLink(id, req.ThreadLink)
			if err != nil {
				logger.Error("api set thread link failed", zap.Int64("id", id), zap.Error(err))
				apiError(w, err.Error(), 400)
				return
			}
			apiResponse(w, tf, "thread updated")
		} else {
			apiError(w, "method not allowed", 405)
		}
	case "tags":
		if len(parts) >= 3 && r.Method == http.MethodDelete {
			tagID, _ := strconv.ParseInt(parts[2], 10, 64)
			tf, err := RemoveTagFromTodo(id, tagID)
			if err != nil {
				logger.Error("api remove tag failed", zap.Int64("tag_id", tagID), zap.Error(err))
				apiError(w, err.Error(), 400)
				return
			}
			apiResponse(w, tf, "tag removed")
		} else if r.Method == http.MethodPost {
			var req struct {
				GroupName string `json:"group_name"`
				Tag       string `json:"tag"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
			json.NewDecoder(r.Body).Decode(&req)
			tf, err := AddTagToTodo(id, req.GroupName, req.Tag)
			if err != nil {
				logger.Error("api add tag failed", zap.Int64("id", id), zap.Error(err))
				apiError(w, err.Error(), 400)
				return
			}
			apiResponse(w, tf, "tag added")
		} else {
			apiError(w, "method not allowed", 405)
		}
	case "schedule":
		if r.Method == http.MethodPost {
			var req struct {
				ScheduleAt int64 `json:"schedule_at"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
			json.NewDecoder(r.Body).Decode(&req)
			tf, err := SetSchedule(id, req.ScheduleAt)
			if err != nil {
				logger.Error("api schedule failed", zap.Int64("id", id), zap.Error(err))
				apiError(w, err.Error(), 400)
				return
			}
			apiResponse(w, tf, "schedule updated")
		} else {
			apiError(w, "method not allowed", 405)
		}
	case "later":
		if r.Method == http.MethodPost {
			var req struct {
				ScheduleAt int64 `json:"schedule_at"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
			json.NewDecoder(r.Body).Decode(&req)
			tf, err := Later(id, req.ScheduleAt)
			if err != nil {
				logger.Error("api later failed", zap.Int64("id", id), zap.Error(err))
				apiError(w, err.Error(), 400)
				return
			}
			apiResponse(w, tf, "postponed")
		} else {
			apiError(w, "method not allowed", 405)
		}
	case "start":
		if r.Method == http.MethodPost {
			tf, err := Start(id)
			if err != nil {
				logger.Error("api start failed", zap.Int64("id", id), zap.Error(err))
				apiError(w, err.Error(), 400)
				return
			}
			apiResponse(w, tf, "started")
		} else {
			apiError(w, "method not allowed", 405)
		}
	case "pause":
		if r.Method == http.MethodPost {
			tf, err := Pause(id)
			if err != nil {
				logger.Error("api pause failed", zap.Int64("id", id), zap.Error(err))
				apiError(w, err.Error(), 400)
				return
			}
			apiResponse(w, tf, "paused")
		} else {
			apiError(w, "method not allowed", 405)
		}
	case "reopen":
		if r.Method == http.MethodPost {
			tf, err := Reopen(id)
			if err != nil {
				logger.Error("api reopen failed", zap.Int64("id", id), zap.Error(err))
				apiError(w, err.Error(), 400)
				return
			}
			apiResponse(w, tf, "reopened")
		} else {
			apiError(w, "method not allowed", 405)
		}
	case "complete":
		if r.Method == http.MethodPost {
			var req struct {
				Note string `json:"note"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
			json.NewDecoder(r.Body).Decode(&req)
			tf, err := Complete(id, req.Note)
			if err != nil {
				logger.Error("api complete failed", zap.Int64("id", id), zap.Error(err))
				apiError(w, err.Error(), 400)
				return
			}
			apiResponse(w, tf, "completed")
		} else {
			apiError(w, "method not allowed", 405)
		}
	default:
		apiError(w, "not found", 404)
	}
}

func handleToken(w http.ResponseWriter, r *http.Request) {
	logger := pkglog.GetLogger(r.Context())
	if r.Method != http.MethodPost {
		apiError(w, "method not allowed", 405)
		return
	}
	if config.Get().HasToken() {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") || !config.Get().VerifyToken(strings.TrimPrefix(authHeader, "Bearer ")) {
			logger.Warn("unauthorized token change attempt")
			apiError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}
	var req struct {
		Token string `json:"token"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("token api: invalid json", zap.Error(err))
		apiError(w, "invalid json", 400)
		return
	}
	if req.Token == "" {
		logger.Error("token api: empty token not allowed")
		apiError(w, "token cannot be empty", 400)
		return
	}
	if err := config.Get().SetToken(req.Token); err != nil {
		logger.Error("token api: failed to save token", zap.Error(err))
		apiError(w, "failed to save token", 500)
		return
	}
	jsonResp(w, map[string]interface{}{"success": true, "message": "token updated"})
}

func jsonResp(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
