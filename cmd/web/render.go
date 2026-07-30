package main

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"
	"sync"
)

// Renderer parses and executes Go HTML templates.
type Renderer struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
}

// NewRenderer parses templates from the embedded filesystem.
// Each page template is combined with its parent layout (base or layout).
func NewRenderer(fsys fs.FS) (*Renderer, error) {
	r := &Renderer{templates: make(map[string]*template.Template)}

	// Pages using base.html (no sidebar): login, register
	for _, page := range []string{"pages/login.html", "pages/register.html"} {
		t, err := template.ParseFS(fsys, "base.html", page)
		if err != nil {
			return nil, err
		}
		r.templates[page] = t
	}

	// Pages using layout.html (with sidebar): dashboard, users
	for _, page := range []string{"pages/dashboard.html", "pages/users.html"} {
		t, err := template.ParseFS(fsys, "layout.html", page)
		if err != nil {
			return nil, err
		}
		r.templates[page] = t
	}

	return r, nil
}

// Render executes a named template and writes the result to w.
func (r *Renderer) Render(w http.ResponseWriter, name string, data any) {
	r.mu.RLock()
	t, ok := r.templates[name]
	r.mu.RUnlock()

	if !ok {
		http.Error(w, "template not found: "+name, 500)
		return
	}

	// Determine the root template name based on layout
	root := "base"
	switch name {
	case "pages/dashboard.html", "pages/users.html":
		root = "layout"
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, root, data); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
