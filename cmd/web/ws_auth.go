package main

import (
	"errors"
	"net/http"
	"strings"
)

// wsTokenProtoPrefix is the agreed Sec-WebSocket-Protocol convention:
// the client sends ["bearer", "<token>"], keeping the token out of the URL.
const wsTokenProtoPrefix = "bearer"

var (
	errMissingToken     = errors.New("缺少认证 token")
	errInvalidTransport = errors.New("token 传输方式无效")
)

// readSubprotocols parses Sec-WebSocket-Protocol header values into a flat
// list, supporting both repeated headers and comma-separated single values.
func readSubprotocols(r *http.Request) []string {
	var out []string
	for _, h := range r.Header.Values("Sec-WebSocket-Protocol") {
		for _, p := range strings.Split(h, ",") {
			if v := strings.TrimSpace(p); v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

// extractWSToken extracts the auth token from a WebSocket handshake request.
// A token present in the URL query is rejected as an invalid transport
// (Requirement 5.4); otherwise the token is read from the Sec-WebSocket-Protocol
// subprotocol list (Requirement 5.2).
func extractWSToken(r *http.Request) (string, error) {
	if r.URL.Query().Get("token") != "" {
		return "", errInvalidTransport
	}
	protos := readSubprotocols(r)
	for i, p := range protos {
		if p == wsTokenProtoPrefix && i+1 < len(protos) {
			if protos[i+1] == "" {
				return "", errMissingToken
			}
			return protos[i+1], nil
		}
	}
	return "", errMissingToken
}
