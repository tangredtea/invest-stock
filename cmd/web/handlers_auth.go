package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"invest/internal/auth"
	"invest/internal/model"
)

type authApp struct {
	jwt     *auth.JWTManager
	users   *model.UserStore
	limiter *LoginRateLimiter
}

type jsonResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (a *authApp) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, jsonResp{Code: 405, Message: "method not allowed"})
		return
	}

	ip := clientIP(r)
	if a.limiter != nil && !a.limiter.Allowed(ip) {
		writeJSON(w, 429, jsonResp{Code: 429, Message: "登录尝试过于频繁,请稍后再试"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "请求格式错误"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "用户名和密码不能为空"})
		return
	}

	user, err := a.users.GetByUsername(req.Username)
	if err != nil || !auth.CheckPassword(user.Password, req.Password) {
		if a.limiter != nil {
			a.limiter.RecordFailure(ip)
		}
		writeJSON(w, 401, jsonResp{Code: 401, Message: "用户名或密码错误"})
		return
	}

	if user.Status != 1 {
		writeJSON(w, 403, jsonResp{Code: 403, Message: "账户已被禁用"})
		return
	}

	if a.limiter != nil {
		a.limiter.Reset(ip)
	}

	token, err := a.jwt.Generate(user.ID, user.Username, user.Role)
	if err != nil {
		writeJSON(w, 500, jsonResp{Code: 500, Message: "生成令牌失败"})
		return
	}

	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: map[string]any{
		"token": token,
		"user":  user,
	}})
}

func (a *authApp) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, jsonResp{Code: 405, Message: "method not allowed"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "请求格式错误"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Password) < 6 {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "用户名至少3位，密码至少6位"})
		return
	}

	user, err := a.users.Create(req.Username, req.Password, "user")
	if err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "用户名已存在"})
		return
	}

	token, _ := a.jwt.Generate(user.ID, user.Username, user.Role)
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: map[string]any{
		"token": token,
		"user":  user,
	}})
}

func (a *authApp) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	user, err := a.users.GetByID(claims.UserID)
	if err != nil {
		writeJSON(w, 404, jsonResp{Code: 404, Message: "用户不存在"})
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: user})
}

func (a *authApp) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.users.List()
	if err != nil {
		writeJSON(w, 500, jsonResp{Code: 500, Message: "查询失败"})
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: users})
}

func (a *authApp) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, 405, jsonResp{Code: 405, Message: "method not allowed"})
		return
	}

	// Extract user ID from path: /api/admin/users/{id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "缺少用户ID"})
		return
	}
	id, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "无效用户ID"})
		return
	}

	// Disallow operating on the currently authenticated account to prevent
	// administrators from locking themselves out (Requirement 9.1).
	if claims := getClaims(r); claims != nil && claims.UserID == id {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "不能修改自己的账户"})
		return
	}

	var req struct {
		Role   string `json:"role"`
		Status int    `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "请求格式错误"})
		return
	}

	if err := a.users.Update(id, req.Role, req.Status); err != nil {
		writeJSON(w, 500, jsonResp{Code: 500, Message: "更新失败"})
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok"})
}

func (a *authApp) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, 405, jsonResp{Code: 405, Message: "method not allowed"})
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "缺少用户ID"})
		return
	}
	id, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "无效用户ID"})
		return
	}

	// Disallow self-deletion (Requirement 9.2).
	if claims := getClaims(r); claims != nil && claims.UserID == id {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "不能删除自己的账户"})
		return
	}

	if err := a.users.Delete(id); err != nil {
		writeJSON(w, 500, jsonResp{Code: 500, Message: "删除失败"})
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok"})
}
