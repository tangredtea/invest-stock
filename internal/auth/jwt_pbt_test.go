package auth

import (
	"errors"
	"math/rand"
	"testing"
	"testing/quick"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// makeClaims builds a Claims value with the given fields and a valid time window.
func makeClaims(uid int64, usr, role string, ttl time.Duration) Claims {
	now := time.Now()
	return Claims{
		UserID:   uid,
		Username: usr,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
}

// Feature: security-and-performance-hardening, Property 1: 对任意 token,当且仅当其头部签名方法属于
// HMAC 族、签名以所配置密钥验证通过且未过期时 Parse 才返回声明且无错误;否则(非 HMAC 算法、none、
// 签名错误、已过期)一律返回无效 token 错误且不返回任何解码声明。
func TestProperty1_AlgorithmAndSignatureValidation(t *testing.T) {
	secret := "this-is-a-sufficiently-long-secret-key-1234567890"
	mgr := NewJWTManager(secret, time.Hour)

	// 1a. Valid HMAC (HS256) tokens with the configured secret must parse.
	validHMAC := func(uid int64, usr, role string) bool {
		tok, err := mgr.Generate(uid, usr, role)
		if err != nil {
			return false
		}
		claims, err := mgr.Parse(tok)
		return err == nil && claims != nil
	}
	if err := quick.Check(validHMAC, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("valid HMAC token should parse: %v", err)
	}

	// 1b. Tokens signed with a different (wrong) secret must be rejected.
	wrongSecret := func(uid int64, usr, role string) bool {
		other := jwt.NewWithClaims(jwt.SigningMethodHS256, makeClaims(uid, usr, role, time.Hour))
		signed, err := other.SignedString([]byte("a-totally-different-secret-key-0987654321"))
		if err != nil {
			return true // signing failure is not the property under test
		}
		claims, err := mgr.Parse(signed)
		return err == ErrInvalidToken && claims == nil
	}
	if err := quick.Check(wrongSecret, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("token signed with wrong secret must be rejected: %v", err)
	}

	// 1c. The "none" algorithm must always be rejected.
	noneAlg := func(uid int64, usr, role string) bool {
		tok := jwt.NewWithClaims(jwt.SigningMethodNone, makeClaims(uid, usr, role, time.Hour))
		signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			return true
		}
		claims, err := mgr.Parse(signed)
		return err == ErrInvalidToken && claims == nil
	}
	if err := quick.Check(noneAlg, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf(`"none" algorithm must be rejected: %v`, err)
	}

	// 1d. Expired HMAC tokens (correct secret) must be rejected with the
	// dedicated expired-token error.
	expired := func(uid int64, usr, role string) bool {
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, makeClaims(uid, usr, role, -time.Hour))
		signed, err := tok.SignedString([]byte(secret))
		if err != nil {
			return true
		}
		claims, err := mgr.Parse(signed)
		return errors.Is(err, ErrExpiredToken) && claims == nil
	}
	if err := quick.Check(expired, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("expired token must be rejected: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 2: 对任意合法的用户标识、用户名与角色,
// 先 Generate 再 Parse 所得声明中的 UserID、Username、Role 与原始输入逐一相等,且不返回错误。
func TestProperty2_GenerateParseRoundTrip(t *testing.T) {
	mgr := NewJWTManager("this-is-a-sufficiently-long-secret-key-1234567890", time.Hour)

	roundTrip := func(uid int64, usr, role string) bool {
		tok, err := mgr.Generate(uid, usr, role)
		if err != nil {
			return false
		}
		claims, err := mgr.Parse(tok)
		if err != nil || claims == nil {
			return false
		}
		return claims.UserID == uid && claims.Username == usr && claims.Role == role
	}
	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(1))}
	if err := quick.Check(roundTrip, cfg); err != nil {
		t.Errorf("generate/parse round-trip must preserve claims: %v", err)
	}
}
