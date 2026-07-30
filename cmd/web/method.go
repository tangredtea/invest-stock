package main

import (
	"net/http"
	"strings"
)

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	return requireAnyMethod(w, r, method)
}

func requireAnyMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeJSON(w, http.StatusMethodNotAllowed, jsonResp{Code: http.StatusMethodNotAllowed, Message: "method not allowed"})
	return false
}
