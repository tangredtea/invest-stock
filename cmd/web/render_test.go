package main

import (
	"html/template"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRendererRenderKnownTemplate(t *testing.T) {
	tmplSub, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		t.Fatalf("load templates: %v", err)
	}
	renderer, err := NewRenderer(tmplSub)
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}

	rec := httptest.NewRecorder()
	renderer.Render(rec, "pages/login.html", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", got)
	}
}

func TestRendererRenderMissingTemplate(t *testing.T) {
	tmplSub, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		t.Fatalf("load templates: %v", err)
	}
	renderer, err := NewRenderer(tmplSub)
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}

	rec := httptest.NewRecorder()
	renderer.Render(rec, "pages/missing.html", nil)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestRendererRenderExecutionErrorDoesNotAdvertiseHTML(t *testing.T) {
	renderer := &Renderer{
		templates: map[string]*template.Template{
			"broken.html": template.Must(template.New("base").Option("missingkey=error").Parse(`{{.Missing}}`)),
		},
	}

	rec := httptest.NewRecorder()
	renderer.Render(rec, "broken.html", map[string]string{})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got == "text/html; charset=utf-8" {
		t.Fatalf("execution errors must not be labelled as rendered HTML")
	}
}
