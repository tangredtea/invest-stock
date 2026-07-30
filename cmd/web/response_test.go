package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSONWritesStatusAndBody(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSON(rec, http.StatusCreated, jsonResp{Code: 0, Message: "ok", Data: map[string]string{"id": "1"}})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var body jsonResp
	decodeTestResponse(t, rec, &body)
	if body.Code != 0 || body.Message != "ok" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestWriteJSONFallsBackWhenEncodingFails(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSON(rec, http.StatusOK, map[string]any{"bad": make(chan int)})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var body jsonResp
	decodeTestResponse(t, rec, &body)
	if body.Code != http.StatusInternalServerError || body.Message == "" {
		t.Fatalf("unexpected fallback body: %+v", body)
	}
}
