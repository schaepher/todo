package app

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
	"runtime"
	"todo/config"
	pkglog "todo/pkg/log"
	"go.uber.org/zap"
)

var tmpl *template.Template

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	tmpl = template.Must(template.ParseGlob(filepath.Join(dir, "..", "templates", "*.html")))
}

func SetupUIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/admin", handleAdminPage)
}

func handleAdminPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tagGroups := buildTagGroupView()
	tagGroupsJSON, _ := json.Marshal(tagGroups)

	data := struct {
		Token          string
		TagGroups      []TagGroupView
		TagGroupsJSON  template.JS
	}{
		Token:          config.Get().TokenValue(),
		TagGroups:      tagGroups,
		TagGroupsJSON:  template.JS(tagGroupsJSON),
	}
	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		pkglog.GetLogger(r.Context()).Error("template execution failed", zap.Error(err))
	}
}

type TagGroupView struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func buildTagGroupView() []TagGroupView {
	cfg := config.Get()
	groups := make([]TagGroupView, len(cfg.TagGroups))
	for i, g := range cfg.TagGroups {
		groups[i] = TagGroupView{Name: g.Name, Tags: g.Tags}
	}
	return groups
}
