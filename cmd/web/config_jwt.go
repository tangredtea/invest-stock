package main

import "errors"

// knownInsecureSecrets are values that must never be used as the JWT secret.
var knownInsecureSecrets = map[string]bool{
	"invest-secret-key-2024": true,
}

// validateJWTSecret reports a fatal error if the secret is missing, too short,
// or a known-insecure default. Exposed for property testing.
func validateJWTSecret(secret string) error {
	if secret == "" {
		return errors.New("INVEST_JWT_SECRET 未设置: 拒绝启动")
	}
	if len(secret) < minJWTSecretLen || knownInsecureSecrets[secret] {
		return errors.New("INVEST_JWT_SECRET 不安全(长度不足 32 或为已知默认值): 拒绝启动")
	}
	return nil
}
