package main

import (
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: security-and-performance-hardening, Property 9: 对任意密钥字符串,LoadConfig(经
// validateJWTSecret)当且仅当其长度小于 32 或等于已知不安全默认值时返回致命错误;长度不小于 32
// 且不在黑名单中的密钥被接受。
func TestProperty9_JWTSecretStrengthGate(t *testing.T) {
	f := func(secret string) bool {
		err := validateJWTSecret(secret)
		shouldReject := len(secret) < minJWTSecretLen || knownInsecureSecrets[secret]
		if shouldReject {
			return err != nil
		}
		return err == nil
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("JWT secret gate violated: %v", err)
	}
}

func TestValidateJWTSecretCases(t *testing.T) {
	cases := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"empty", "", true},
		{"too short", "short-secret", true},
		{"known insecure default", "invest-secret-key-2024", true},
		{"exactly 32 chars", strings.Repeat("a", 32), false},
		{"long strong secret", strings.Repeat("x", 64), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := validateJWTSecret(c.secret) != nil; got != c.wantErr {
				t.Errorf("validateJWTSecret(%q) err=%v, wantErr=%v", c.secret, got, c.wantErr)
			}
		})
	}
}

func TestLoadConfigMissingSecretIsFatal(t *testing.T) {
	t.Setenv("INVEST_JWT_SECRET", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected fatal error when JWT secret is missing")
	}
}

func TestLoadConfigSkipsSeedingWhenAdminMissing(t *testing.T) {
	t.Setenv("INVEST_JWT_SECRET", strings.Repeat("s", 40))
	t.Setenv("INVEST_ADMIN_USERNAME", "")
	t.Setenv("INVEST_ADMIN_PASSWORD", "")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SeedAdmin {
		t.Error("expected SeedAdmin=false when admin credentials are absent")
	}
}

func TestLoadConfigSkipsSeedingWhenPasswordTooShort(t *testing.T) {
	t.Setenv("INVEST_JWT_SECRET", strings.Repeat("s", 40))
	t.Setenv("INVEST_ADMIN_USERNAME", "admin")
	t.Setenv("INVEST_ADMIN_PASSWORD", "short") // < 8 chars
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SeedAdmin {
		t.Error("expected SeedAdmin=false when admin password is too short")
	}
}

func TestLoadConfigSeedsWhenAdminValid(t *testing.T) {
	t.Setenv("INVEST_JWT_SECRET", strings.Repeat("s", 40))
	t.Setenv("INVEST_ADMIN_USERNAME", "admin")
	t.Setenv("INVEST_ADMIN_PASSWORD", "strongpass")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.SeedAdmin {
		t.Error("expected SeedAdmin=true with valid admin credentials")
	}
}

func TestLoadConfigTrimsEnvironmentValues(t *testing.T) {
	secret := strings.Repeat("s", 40)
	t.Setenv("INVEST_JWT_SECRET", " \n"+secret+"\t ")
	t.Setenv("INVEST_ADDR", " \t:9090\n")
	t.Setenv("INVEST_ADMIN_USERNAME", " admin ")
	t.Setenv("INVEST_ADMIN_PASSWORD", "\tstrongpass\n")
	t.Setenv("INVEST_KLINE_TTL", " 2h ")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.JWTSecret != secret {
		t.Errorf("expected trimmed JWT secret %q, got %q", secret, cfg.JWTSecret)
	}
	if cfg.Addr != ":9090" {
		t.Errorf("expected trimmed addr :9090, got %q", cfg.Addr)
	}
	if cfg.KLineTTL != 2*time.Hour {
		t.Errorf("expected trimmed TTL 2h, got %v", cfg.KLineTTL)
	}
	if !cfg.SeedAdmin {
		t.Fatal("expected trimmed admin credentials to be valid")
	}
	if cfg.AdminUsername != "admin" {
		t.Errorf("expected trimmed admin username, got %q", cfg.AdminUsername)
	}
	if cfg.AdminPassword != "strongpass" {
		t.Errorf("expected trimmed admin password, got %q", cfg.AdminPassword)
	}
}

func TestLoadConfigRejectsPaddedInsecureDefaultSecret(t *testing.T) {
	t.Setenv("INVEST_JWT_SECRET", " invest-secret-key-2024 ")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected fatal error when padded JWT secret is a known insecure default")
	}
}

func TestLoadConfigWhitespaceAddrFallsBackToDefault(t *testing.T) {
	t.Setenv("INVEST_JWT_SECRET", strings.Repeat("s", 40))
	t.Setenv("INVEST_ADDR", " \t\n")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Addr != defaultAddr {
		t.Errorf("expected default addr %q, got %q", defaultAddr, cfg.Addr)
	}
}

func TestParseKLineTTLClampingAndDefault(t *testing.T) {
	if got := parseKLineTTL(""); got != defaultKLineTTL {
		t.Errorf("empty TTL should default to 24h, got %v", got)
	}
	if got := parseKLineTTL("500ms"); got != minConfigKLineTTL {
		t.Errorf("sub-second TTL should clamp to 1s, got %v", got)
	}
	if got := parseKLineTTL("100h"); got != maxConfigKLineTTL {
		t.Errorf("over-max TTL should clamp to 24h, got %v", got)
	}
	if got := parseKLineTTL("2h"); got != 2*time.Hour {
		t.Errorf("in-range TTL should be preserved, got %v", got)
	}
	if got := parseKLineTTL("not-a-duration"); got != defaultKLineTTL {
		t.Errorf("invalid TTL should default to 24h, got %v", got)
	}
}

func TestLoadConfigAddrDefault(t *testing.T) {
	t.Setenv("INVEST_JWT_SECRET", strings.Repeat("s", 40))
	t.Setenv("INVEST_ADDR", "")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Addr != defaultAddr {
		t.Errorf("expected default addr %q, got %q", defaultAddr, cfg.Addr)
	}
}
