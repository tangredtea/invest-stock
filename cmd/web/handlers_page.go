package main

import (
	"net/http"
)

// pageApp holds dependencies for page rendering.
type pageApp struct {
	renderer *Renderer
}

func (p *pageApp) handleLogin(w http.ResponseWriter, r *http.Request) {
	p.renderer.Render(w, "pages/login.html", nil)
}

func (p *pageApp) handleRegister(w http.ResponseWriter, r *http.Request) {
	p.renderer.Render(w, "pages/register.html", nil)
}

func (p *pageApp) handleDashboard(w http.ResponseWriter, r *http.Request) {
	p.renderer.Render(w, "pages/dashboard.html", nil)
}

func (p *pageApp) handleUsers(w http.ResponseWriter, r *http.Request) {
	p.renderer.Render(w, "pages/users.html", nil)
}
