package render

import (
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
)

// Renderer parses and executes HTML templates from an embedded filesystem.
type Renderer struct {
	templates map[string]*template.Template
}

// NewRenderer parses all templates from the given filesystem.
// Each page template is combined with the base layout and all partials.
func NewRenderer(fsys fs.FS) *Renderer {
	r := &Renderer{
		templates: make(map[string]*template.Template),
	}

	// Collect partials
	partials, err := fs.Glob(fsys, "partials/*.html")
	if err != nil {
		slog.Error("failed to glob partials", "error", err)
	}

	// Collect page templates (top-level *.html except base.html)
	pages, err := fs.Glob(fsys, "*.html")
	if err != nil {
		slog.Error("failed to glob pages", "error", err)
		return r
	}

	// Build template set for each page
	for _, page := range pages {
		name := filepath.Base(page)
		if name == "base.html" {
			continue
		}

		files := []string{"base.html"}
		files = append(files, partials...)
		files = append(files, page)

		tmpl, err := template.New("").ParseFS(fsys, files...)
		if err != nil {
			slog.Error("failed to parse template", "page", name, "error", err)
			continue
		}
		r.templates[name] = tmpl
	}

	return r
}

// Render executes the named template with the given data.
// For HTMX partial requests (HX-Request header), it executes just the "content"
// block. For full page requests, it executes the "base" template.
// It automatically injects the CSRF token from the cookie into template data.
func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, tmpl string, data map[string]interface{}) {
	t, ok := r.templates[tmpl]
	if !ok {
		slog.Error("template not found", "name", tmpl)
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	// Inject CSRF token from cookie so templates can reference {{.CSRFToken}}
	if cookie, err := req.Cookie("csrf_token"); err == nil {
		data["CSRFToken"] = cookie.Value
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	blockName := "base"
	if strings.ToLower(req.Header.Get("HX-Request")) == "true" {
		blockName = "content"
	}

	if err := t.ExecuteTemplate(w, blockName, data); err != nil {
		slog.Error("failed to execute template", "name", tmpl, "block", blockName, "error", err)
	}
}
