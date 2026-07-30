package main

import (
	"errors"
	"strconv"
	"strings"
)

type authCredentialsReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type adminUpdateUserReq struct {
	Role   string `json:"role"`
	Status int    `json:"status"`
}

func adminUserIDFromPath(path string) (int64, error) {
	parts := strings.Split(path, "/")
	if len(parts) != 5 || parts[1] != "api" || parts[2] != "admin" || parts[3] != "users" {
		return 0, errors.New("无效用户路径")
	}
	if parts[4] == "" {
		return 0, errors.New("缺少用户ID")
	}
	id, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("无效用户ID")
	}
	return id, nil
}
