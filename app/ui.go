package app

import (
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"todo/config"
	pkglog "todo/pkg/log"
	"go.uber.org/zap"
)

//go:embed templates/*.html
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

func SetupUIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/admin", handleAdminPage)
}

func handleAdminPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tagGroups := buildTagGroupView()
	tagGroupsJSON, _ := json.Marshal(tagGroups)
	safeJSON := strings.ReplaceAll(string(tagGroupsJSON), "</", "\\u003C/")

	data := struct {
		Token          string
		TagGroups      []TagGroupView
		TagGroupsJSON  template.JS
	}{
		Token:          config.Get().TokenValue(),
		TagGroups:      tagGroups,
		TagGroupsJSON:  template.JS(safeJSON),
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
