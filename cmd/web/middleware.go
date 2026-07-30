package main

import (
	"context"
	"errors"
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

		if tokenStr == "" {
			writeJSON(w, http.StatusUnauthorized, jsonResp{Code: http.StatusUnauthorized, Message: "未登录"})
			return
		}

		claims, err := jwtMgr.Parse(tokenStr)
		if err != nil {
			msg := "认证失败"
			if errors.Is(err, auth.ErrExpiredToken) {
				msg = "登录已过期"
			}
			writeJSON(w, http.StatusUnauthorized, jsonResp{Code: http.StatusUnauthorized, Message: msg})
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
			writeJSON(w, http.StatusForbidden, jsonResp{Code: http.StatusForbidden, Message: "需要管理员权限"})
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
