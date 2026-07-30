package main

import (
	"errors"
	"net/http"

	"invest/internal/auth"
	"invest/internal/model"
)

type authApp struct {
	jwt     *auth.JWTManager
	users   *model.UserStore
	limiter *LoginRateLimiter
}

func (a *authApp) handleMe(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	claims := getClaims(r)
	user, err := a.users.GetByID(claims.UserID)
	if err != nil {
		writeJSON(w, 404, jsonResp{Code: 404, Message: "用户不存在"})
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: user})
}

func (a *authApp) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	users, err := a.users.List()
	if err != nil {
		writeJSON(w, 500, jsonResp{Code: 500, Message: "查询失败"})
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: users})
}

func (a *authApp) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPut) {
		return
	}

	id, err := adminUserIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: err.Error()})
		return
	}

	// Disallow operating on the currently authenticated account to prevent
	// administrators from locking themselves out (Requirement 9.1).
	if claims := getClaims(r); claims != nil && claims.UserID == id {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "不能修改自己的账户"})
		return
	}

	var req adminUpdateUserReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "请求格式错误"})
		return
	}

	if err := a.users.Update(id, req.Role, req.Status); err != nil {
		switch {
		case errors.Is(err, model.ErrInvalidRole), errors.Is(err, model.ErrInvalidStatus):
			writeJSON(w, 400, jsonResp{Code: 400, Message: "角色或状态无效"})
		case errors.Is(err, model.ErrUserNotFound):
			writeJSON(w, 404, jsonResp{Code: 404, Message: "用户不存在"})
		default:
			writeJSON(w, 500, jsonResp{Code: 500, Message: "更新失败"})
		}
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok"})
}

func (a *authApp) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}

	id, err := adminUserIDFromPath(r.URL.Path)
	if err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: err.Error()})
		return
	}

	// Disallow self-deletion (Requirement 9.2).
	if claims := getClaims(r); claims != nil && claims.UserID == id {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "不能删除自己的账户"})
		return
	}

	if err := a.users.Delete(id); err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			writeJSON(w, 404, jsonResp{Code: 404, Message: "用户不存在"})
			return
		}
		writeJSON(w, 500, jsonResp{Code: 500, Message: "删除失败"})
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok"})
}
