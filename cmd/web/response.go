package main

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type jsonResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		status = http.StatusInternalServerError
		buf.Reset()
		_ = json.NewEncoder(&buf).Encode(jsonResp{Code: status, Message: "响应编码失败"})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
