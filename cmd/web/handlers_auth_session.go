package main

import (
	"errors"
	"net/http"
	"strings"

	"invest/internal/auth"
	"invest/internal/model"
)

func (a *authApp) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	ip := clientIP(r)
	if a.limiter != nil && !a.limiter.Allowed(ip) {
		writeJSON(w, 429, jsonResp{Code: 429, Message: "登录尝试过于频繁,请稍后再试"})
		return
	}

	var req authCredentialsReq
	if err := decodeJSON(w, r, &req); err != nil {
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

	a.writeAuthSuccess(w, user)
}

func (a *authApp) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req authCredentialsReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "请求格式错误"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Password) < 6 {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "用户名至少3位，密码至少6位"})
		return
	}

	user, err := a.users.Create(req.Username, req.Password, model.RoleUser)
	if err != nil {
		if errors.Is(err, model.ErrUserExists) {
			writeJSON(w, 400, jsonResp{Code: 400, Message: "用户名已存在"})
			return
		}
		writeJSON(w, 500, jsonResp{Code: 500, Message: "注册失败"})
		return
	}

	a.writeAuthSuccess(w, user)
}

func (a *authApp) writeAuthSuccess(w http.ResponseWriter, user *model.User) {
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
