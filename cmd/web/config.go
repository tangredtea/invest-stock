package main

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// Config holds runtime configuration sourced from environment variables.
type Config struct {
	JWTSecret     string        // INVEST_JWT_SECRET (required, >=32 chars, not a known-insecure default)
	AdminUsername string        // INVEST_ADMIN_USERNAME (1..64); empty/invalid => skip seeding
	AdminPassword string        // INVEST_ADMIN_PASSWORD (8..128); empty/invalid => skip seeding
	Addr          string        // INVEST_ADDR (default ":8081")
	JWTTTL        time.Duration // fixed 24h, preserves current behavior
	KLineTTL      time.Duration // INVEST_KLINE_TTL (default 24h, clamped to [1s,24h])
	SeedAdmin     bool          // whether admin credentials are valid for seeding
}

const (
	minJWTSecretLen   = 32
	minAdminUserLen   = 1
	maxAdminUserLen   = 64
	minAdminPassLen   = 8
	maxAdminPassLen   = 128
	defaultAddr       = ":8081"
	defaultKLineTTL   = 24 * time.Hour
	minConfigKLineTTL = 1 * time.Second
	maxConfigKLineTTL = 24 * time.Hour
)

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

// LoadConfig reads and validates configuration from the environment. A missing
// or insecure JWT secret is fatal (returns an error so the caller can exit
// before binding a port). Missing/invalid admin credentials are non-fatal:
// seeding is skipped and SeedAdmin is set to false.
func LoadConfig() (*Config, error) {
	secret := os.Getenv("INVEST_JWT_SECRET")
	if err := validateJWTSecret(secret); err != nil {
		return nil, err
	}

	cfg := &Config{
		JWTSecret: secret,
		Addr:      defaultAddr,
		JWTTTL:    24 * time.Hour,
		KLineTTL:  defaultKLineTTL,
	}

	if addr := os.Getenv("INVEST_ADDR"); addr != "" {
		cfg.Addr = addr
	}

	cfg.KLineTTL = parseKLineTTL(os.Getenv("INVEST_KLINE_TTL"))

	user := os.Getenv("INVEST_ADMIN_USERNAME")
	pass := os.Getenv("INVEST_ADMIN_PASSWORD")
	if validAdminUsername(user) && validAdminPassword(pass) {
		cfg.AdminUsername = user
		cfg.AdminPassword = pass
		cfg.SeedAdmin = true
	} else {
		cfg.SeedAdmin = false
		fmt.Println("提示: 初始管理员凭据缺失或不合法,跳过默认管理员播种")
	}

	return cfg, nil
}

func validAdminUsername(u string) bool {
	return len(u) >= minAdminUserLen && len(u) <= maxAdminUserLen
}

func validAdminPassword(p string) bool {
	return len(p) >= minAdminPassLen && len(p) <= maxAdminPassLen
}

// parseKLineTTL parses INVEST_KLINE_TTL, defaulting to 24h and clamping to
// [1s, 24h]. Invalid values fall back to the default.
func parseKLineTTL(raw string) time.Duration {
	if raw == "" {
		return defaultKLineTTL
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		fmt.Printf("提示: INVEST_KLINE_TTL 无法解析(%q),使用默认 24h\n", raw)
		return defaultKLineTTL
	}
	if d < minConfigKLineTTL {
		return minConfigKLineTTL
	}
	if d > maxConfigKLineTTL {
		return maxConfigKLineTTL
	}
	return d
}
