package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func callTestHandler(method, target, body string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	req := jsonTestRequest(method, target, body)
	return recordTestRequest(req, handler)
}

func jsonTestRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func recordTestRequest(req *http.Request, handler http.HandlerFunc) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func postTestJSON(target, body string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	return callTestHandler(http.MethodPost, target, body, handler)
}

func decodeTestResponse(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
