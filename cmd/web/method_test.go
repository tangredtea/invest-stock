package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireMethodRejectsUnexpectedMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/resource", nil)
	rec := httptest.NewRecorder()

	if requireMethod(rec, req, http.MethodGet) {
		t.Fatal("requireMethod accepted unexpected method")
	}
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", got, http.MethodGet)
	}
}

func TestRequireMethodAcceptsExpectedMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	rec := httptest.NewRecorder()

	if !requireMethod(rec, req, http.MethodGet) {
		t.Fatal("requireMethod rejected expected method")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want default 200 before write", rec.Code)
	}
}

func TestRequireAnyMethodRejectsUnexpectedMethodWithAllowHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/resource/1", nil)
	rec := httptest.NewRecorder()

	if requireAnyMethod(rec, req, http.MethodPut, http.MethodDelete) {
		t.Fatal("requireAnyMethod accepted unexpected method")
	}
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "PUT, DELETE" {
		t.Fatalf("Allow = %q, want %q", got, "PUT, DELETE")
	}
}

func TestRequireAnyMethodAcceptsOneOfAllowedMethods(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/resource/1", nil)
	rec := httptest.NewRecorder()

	if !requireAnyMethod(rec, req, http.MethodPut, http.MethodDelete) {
		t.Fatal("requireAnyMethod rejected allowed method")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want default 200 before write", rec.Code)
	}
}
