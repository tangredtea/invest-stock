package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"alice","extra":1}`))
	rec := httptest.NewRecorder()

	if err := decodeJSON(rec, req, &dst); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestDecodeJSONRejectsMultipleValues(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"alice"}{"name":"bob"}`))
	rec := httptest.NewRecorder()

	if err := decodeJSON(rec, req, &dst); err == nil {
		t.Fatal("expected multiple JSON values error")
	}
}

func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	var dst map[string]string
	body := `{"x":"` + strings.Repeat("a", maxJSONBodyBytes) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	rec := httptest.NewRecorder()

	if err := decodeJSON(rec, req, &dst); err == nil {
		t.Fatal("expected oversized body error")
	}
}
