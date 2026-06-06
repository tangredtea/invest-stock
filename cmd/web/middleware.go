package main

import (
	"context"
	"net/http"
	"strings"

	"invest/internal/auth"
)

type ctxKey string

const claimsKey ctxKey = "claims"

// requireAuth checks for a valid Bearer token in the Authorization header.
func requireAuth(jwtMgr *auth.JWTManager, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := ""

		// Try Authorization header first
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			tokenStr = h[7:]
		}

		// Fallback to query param (for WebSocket)
		if tokenStr == "" {
			tokenStr = r.URL.Query().Get("token")
		}

		if tokenStr == "" {
			http.Error(w, `{"code":401,"message":"未登录"}`, http.StatusUnauthorized)
			return
		}

		claims, err := jwtMgr.Parse(tokenStr)
		if err != nil {
			http.Error(w, `{"code":401,"message":"登录已过期"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// requireAdmin wraps requireAuth and additionally checks for admin role.
func requireAdmin(jwtMgr *auth.JWTManager, next http.HandlerFunc) http.HandlerFunc {
	return requireAuth(jwtMgr, func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		if claims.Role != "admin" {
			http.Error(w, `{"code":403,"message":"需要管理员权限"}`, http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

// getClaims extracts JWT claims from the request context.
func getClaims(r *http.Request) *auth.Claims {
	c, _ := r.Context().Value(claimsKey).(*auth.Claims)
	return c
}
