# Security Notes

This document records security-relevant design decisions and known residual
risks for the `invest` web application.

## Configuration (required environment variables)

The server refuses to start without a strong JWT secret. Configure these
before running `cmd/web`:

| Variable                | Required | Default   | Constraint                          |
|------------------------|----------|-----------|--------------------------------------|
| `INVEST_JWT_SECRET`    | yes      | —         | length ≥ 32; must not equal known insecure defaults (e.g. `invest-secret-key-2024`) |
| `INVEST_ADMIN_USERNAME`| no       | —         | 1..64 chars; missing/invalid → admin seeding skipped |
| `INVEST_ADMIN_PASSWORD`| no       | —         | 8..128 chars; missing/invalid → admin seeding skipped |
| `INVEST_ADDR`          | no       | `:8081`   | listen address                       |
| `INVEST_KLINE_TTL`     | no       | `24h`     | clamped to `[1s, 24h]`               |

Secrets are never logged. JWT secret rotation invalidates all previously
issued tokens (expected behavior).

## JWT signing

- Tokens are signed with HMAC (HS256) and validated against the configured
  secret. The parser asserts the signing method belongs to the HMAC family,
  so algorithm-confusion attacks (e.g. `alg=none`, asymmetric algorithms) are
  rejected before any claim is trusted.
- Default token TTL is 24 hours.

## Login rate limiting

Failed login attempts are throttled per client IP. The default policy is
**5 failures per 900 seconds** (15 minutes). On the threshold, further
requests from that IP are answered with HTTP 429 without credential
validation. A successful login clears the failure counter for the IP.

X-Forwarded-For is intentionally **not** trusted by default. If you deploy
behind a trusted reverse proxy, adapt `clientIP` to read the forwarded header.

## WebSocket monitor authentication

`/ws/monitor` requires a valid JWT. The token is carried in the
`Sec-WebSocket-Protocol` header as the value following the `bearer`
subprotocol marker:

```js
new WebSocket(url, ['bearer', token]);
```

A token in the URL query string is rejected as an invalid transport — this
keeps the token out of standard HTTP server access logs and browser history.

## Token storage in the browser (residual risk)

The frontend stores the JWT in `localStorage` under the key `token`. This
mechanism has the following exposure:

- **XSS impact**: Any successful cross-site script injection on this origin
  can read `localStorage` and steal the token. Subprotocol-based transport
  removes URL/log leakage, but it does **not** mitigate XSS-driven theft.
- **No `HttpOnly` protection**: `localStorage` is by design accessible to
  JavaScript on the same origin.

### Mitigation directions for future iterations

- **Move the token to an `HttpOnly`, `Secure`, `SameSite=Strict` cookie.**
  Pair with a CSRF defense (e.g. double-submit token or `SameSite` alone for
  same-site flows). This makes the token unreadable to JavaScript.
- **Add a Content Security Policy (CSP)** that restricts script sources, to
  shrink the XSS injection surface.
- **Shorten token TTL and introduce refresh tokens**, so a stolen access
  token is useful for a smaller window.
- **Strict output escaping** on all user-controlled content rendered into
  HTML, to remove XSS injection points at the source.

## Server hardening

- HTTP server enforces read/write/idle timeouts (15s/15s/60s) and a 5s
  read-header timeout to limit slow-client attacks.
- The process performs graceful shutdown on SIGINT/SIGTERM with a 30s
  in-flight grace period and a further 5s force-close window.
- Outbound HTTP calls to upstream APIs use a 10-second client timeout with
  bounded retry (up to 3 attempts, linear backoff).

## Operational guidance

- Run with `GOOS`-appropriate notification handling: desktop notifications
  are macOS-only and degrade silently on other platforms; the monitor stream
  is unaffected.
- Keep the SQLite database file (`data/invest.db*`) out of public storage
  and back it up before changing the JWT secret.
