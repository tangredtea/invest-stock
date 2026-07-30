package main

import (
	"fmt"
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

// LoadConfig reads and validates configuration from the environment. A missing
// or insecure JWT secret is fatal (returns an error so the caller can exit
// before binding a port). Missing/invalid admin credentials are non-fatal:
// seeding is skipped and SeedAdmin is set to false.
func LoadConfig() (*Config, error) {
	secret := envTrimmed("INVEST_JWT_SECRET")
	if err := validateJWTSecret(secret); err != nil {
		return nil, err
	}

	cfg := &Config{
		JWTSecret: secret,
		Addr:      defaultAddr,
		JWTTTL:    24 * time.Hour,
		KLineTTL:  defaultKLineTTL,
	}

	if addr := envTrimmed("INVEST_ADDR"); addr != "" {
		cfg.Addr = addr
	}

	cfg.KLineTTL = parseKLineTTL(envTrimmed("INVEST_KLINE_TTL"))

	user := envTrimmed("INVEST_ADMIN_USERNAME")
	pass := envTrimmed("INVEST_ADMIN_PASSWORD")
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
