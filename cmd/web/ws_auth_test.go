package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/quick"
	"time"

	"invest/internal/auth"
)

// handshakeWithSubprotocols builds a fake WS handshake request carrying the
// given Sec-WebSocket-Protocol values.
func handshakeWithSubprotocols(protos ...string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/ws/monitor?code=600519", nil)
	for _, p := range protos {
		r.Header.Add("Sec-WebSocket-Protocol", p)
	}
	return r
}

// Feature: security-and-performance-hardening, Property 4: 对任意非空 token 字符串,当其作为
// Sec-WebSocket-Protocol 子协议(["bearer", token])随握手传入时,extractWSToken 应返回与原始
// token 逐字符相等的值。
func TestProperty4_SubprotocolTokenRoundTrip(t *testing.T) {
	f := func(token string) bool {
		if token == "" {
			return true // empty token is out of scope for this property
		}
		// Subprotocol tokens cannot contain commas/whitespace per header rules;
		// skip such generated values.
		for _, c := range token {
			if c == ',' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				return true
			}
		}
		r := handshakeWithSubprotocols(wsTokenProtoPrefix, token)
		got, err := extractWSToken(r)
		return err == nil && got == token
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("subprotocol token must round-trip: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 5: 对任意非空 token,当其出现在请求 URL 的
// 查询字符串(?token=...)中时,extractWSToken 应返回“传输方式无效”错误,不据其建立连接。
func TestProperty5_URLTokenRejected(t *testing.T) {
	f := func(token string) bool {
		if token == "" {
			return true
		}
		r := httptest.NewRequest(http.MethodGet, "/ws/monitor?code=600519&token=x", nil)
		r.Header.Add("Sec-WebSocket-Protocol", wsTokenProtoPrefix)
		r.Header.Add("Sec-WebSocket-Protocol", token)
		_, err := extractWSToken(r)
		return err == errInvalidTransport
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("URL-carried token must be rejected: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 3: 对任意 token(有效/无效/过期/非 HMAC
// 算法),WebSocket 监控端点的接受/拒绝判定结果与受保护 REST 端点认证中间件的判定结果相同(二者共用
// 同一 JWTManager.Parse)。
func TestProperty3_WSRestAuthConsistency(t *testing.T) {
	mgr := auth.NewJWTManager("this-is-a-sufficiently-long-secret-key-1234567890", time.Hour)
	validToken, _ := mgr.Generate(1, "alice", "user")

	f := func(useValid bool, garbage string) bool {
		token := garbage
		if useValid {
			token = validToken
		}
		// REST decision: requireAuth accepts iff Parse succeeds.
		_, restErr := mgr.Parse(token)
		restAccepts := restErr == nil

		// WS decision: with the token supplied via subprotocol, the handler
		// accepts iff Parse succeeds on the extracted token.
		r := handshakeWithSubprotocols(wsTokenProtoPrefix, token)
		extracted, exErr := extractWSToken(r)
		wsAccepts := false
		if exErr == nil {
			_, perr := mgr.Parse(extracted)
			wsAccepts = perr == nil
		}
		return restAccepts == wsAccepts
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("WS and REST auth decisions must agree: %v", err)
	}
}

func TestExtractWSTokenMissing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/ws/monitor?code=600519", nil)
	if _, err := extractWSToken(r); err != errMissingToken {
		t.Errorf("expected errMissingToken, got %v", err)
	}
}
