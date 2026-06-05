package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ── In-memory storage ─────────────────────────────────────────────────────────

var (
	authCodes     sync.Map // string → authCode
	refreshTokens sync.Map // string → storedRefreshToken
	sessions      sync.Map // string → sessionData
)

type authCode struct {
	clientID      string
	redirectURI   string
	codeChallenge string
	scopes        string
	userID        string
	expiresAt     time.Time
}

type storedRefreshToken struct {
	userID string
	scopes string
}

type sessionData struct {
	username  string
	expiresAt time.Time
}

// ── Config ────────────────────────────────────────────────────────────────────

type oauthConfig struct {
	serverURL     string
	clientID      string
	clientSecret  string
	jwtSecret     []byte
	adminUser     string
	adminPassHash string
}

var allowedRedirects = []string{
	"https://claude.ai/api/mcp/auth_callback",
	"https://claude.ai/api/auth/callback",
	"http://localhost",
}

func isAllowedRedirectURI(uri string) bool {
	for _, allowed := range allowedRedirects {
		if uri == allowed || strings.HasPrefix(uri, "http://localhost:") || strings.HasPrefix(uri, "http://localhost/") {
			return true
		}
	}
	return false
}

// ── Discovery ─────────────────────────────────────────────────────────────────

func authServerMetadataHandler(cfg oauthConfig) http.HandlerFunc {
	body, _ := json.Marshal(map[string]any{
		"issuer":                                cfg.serverURL,
		"authorization_endpoint":                cfg.serverURL + "/oauth/authorize",
		"token_endpoint":                        cfg.serverURL + "/oauth/token",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":       []string{"S256"},
		"token_endpoint_auth_methods_supported":  []string{"client_secret_post"},
		"scopes_supported":                      []string{"tickets:read", "tickets:write"},
	})
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(body)
	}
}

// ── Authorize ─────────────────────────────────────────────────────────────────

func authorizeHandler(cfg oauthConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		clientID := q.Get("client_id")
		redirectURI := q.Get("redirect_uri")
		responseType := q.Get("response_type")
		state := q.Get("state")
		codeChallenge := q.Get("code_challenge")
		codeChallengeMethod := q.Get("code_challenge_method")
		scope := q.Get("scope")

		if clientID != cfg.clientID {
			http.Error(w, "invalid client_id", http.StatusBadRequest)
			return
		}
		if !isAllowedRedirectURI(redirectURI) {
			http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
			return
		}
		if responseType != "code" {
			http.Error(w, "unsupported response_type", http.StatusBadRequest)
			return
		}
		if codeChallengeMethod != "S256" {
			http.Error(w, "unsupported code_challenge_method — S256 required", http.StatusBadRequest)
			return
		}

		// Reuse existing session if valid
		if cookie, err := r.Cookie("triage_mcp_session"); err == nil {
			if val, ok := sessions.Load(cookie.Value); ok {
				s := val.(sessionData)
				if time.Now().Before(s.expiresAt) {
					issueCodeAndRedirect(w, r, clientID, redirectURI, state, codeChallenge, scope, s.username)
					return
				}
			}
		}

		renderLoginPage(w, loginPageData{
			ClientID:      clientID,
			RedirectURI:   redirectURI,
			State:         state,
			CodeChallenge: codeChallenge,
			Scope:         scope,
		})
	}
}

// ── Login ─────────────────────────────────────────────────────────────────────

func loginHandler(cfg oauthConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		data := loginPageData{
			ClientID:      r.FormValue("client_id"),
			RedirectURI:   r.FormValue("redirect_uri"),
			State:         r.FormValue("state"),
			CodeChallenge: r.FormValue("code_challenge"),
			Scope:         r.FormValue("scope"),
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		if username != cfg.adminUser ||
			bcrypt.CompareHashAndPassword([]byte(cfg.adminPassHash), []byte(password)) != nil {
			data.Error = "Invalid username or password."
			renderLoginPage(w, data)
			return
		}

		// Create session
		sessID := uuid.New().String()
		sessions.Store(sessID, sessionData{
			username:  username,
			expiresAt: time.Now().Add(24 * time.Hour),
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "triage_mcp_session",
			Value:    sessID,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(24 * time.Hour),
		})

		issueCodeAndRedirect(w, r, data.ClientID, data.RedirectURI, data.State, data.CodeChallenge, data.Scope, username)
	}
}

func issueCodeAndRedirect(w http.ResponseWriter, r *http.Request, clientID, redirectURI, state, codeChallenge, scope, username string) {
	b := make([]byte, 32)
	rand.Read(b)
	code := base64.RawURLEncoding.EncodeToString(b)

	authCodes.Store(code, authCode{
		clientID:      clientID,
		redirectURI:   redirectURI,
		codeChallenge: codeChallenge,
		scopes:        scope,
		userID:        username,
		expiresAt:     time.Now().Add(10 * time.Minute),
	})

	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("code", code)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// ── Token ─────────────────────────────────────────────────────────────────────

func tokenHandler(cfg oauthConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			writeTokenError(w, "invalid_request", "could not parse form")
			return
		}

		if r.FormValue("client_id") != cfg.clientID || r.FormValue("client_secret") != cfg.clientSecret {
			writeTokenError(w, "invalid_client", "invalid client credentials")
			return
		}

		switch r.FormValue("grant_type") {
		case "authorization_code":
			exchangeAuthCode(w, r, cfg)
		case "refresh_token":
			exchangeRefreshToken(w, r, cfg)
		default:
			writeTokenError(w, "unsupported_grant_type", "unsupported grant_type")
		}
	}
}

func exchangeAuthCode(w http.ResponseWriter, r *http.Request, cfg oauthConfig) {
	code := r.FormValue("code")
	codeVerifier := r.FormValue("code_verifier")
	redirectURI := r.FormValue("redirect_uri")

	val, ok := authCodes.LoadAndDelete(code)
	if !ok {
		writeTokenError(w, "invalid_grant", "invalid or expired code")
		return
	}
	ac := val.(authCode)

	if time.Now().After(ac.expiresAt) {
		writeTokenError(w, "invalid_grant", "code expired")
		return
	}
	if ac.redirectURI != redirectURI {
		writeTokenError(w, "invalid_grant", "redirect_uri mismatch")
		return
	}

	// Verify PKCE S256
	h := sha256.Sum256([]byte(codeVerifier))
	if base64.RawURLEncoding.EncodeToString(h[:]) != ac.codeChallenge {
		writeTokenError(w, "invalid_grant", "code_verifier mismatch")
		return
	}

	at, err := newAccessToken(cfg, ac.userID, ac.scopes)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	rt := newRefreshToken(ac.userID, ac.scopes)
	writeTokenResponse(w, at, rt, ac.scopes)
}

func exchangeRefreshToken(w http.ResponseWriter, r *http.Request, cfg oauthConfig) {
	rtVal := r.FormValue("refresh_token")

	val, ok := refreshTokens.LoadAndDelete(rtVal)
	if !ok {
		writeTokenError(w, "invalid_grant", "invalid refresh token")
		return
	}
	rt := val.(storedRefreshToken)

	at, err := newAccessToken(cfg, rt.userID, rt.scopes)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	newRT := newRefreshToken(rt.userID, rt.scopes)
	writeTokenResponse(w, at, newRT, rt.scopes)
}

func newAccessToken(cfg oauthConfig, userID, scopes string) (string, error) {
	claims := jwt.MapClaims{
		"sub":    userID,
		"iss":    cfg.serverURL,
		"aud":    jwt.ClaimStrings{cfg.serverURL},
		"exp":    time.Now().Add(time.Hour).Unix(),
		"iat":    time.Now().Unix(),
		"scopes": scopes,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(cfg.jwtSecret)
}

func newRefreshToken(userID, scopes string) string {
	b := make([]byte, 32)
	rand.Read(b)
	rt := base64.RawURLEncoding.EncodeToString(b)
	refreshTokens.Store(rt, storedRefreshToken{userID: userID, scopes: scopes})
	return rt
}

func writeTokenResponse(w http.ResponseWriter, accessToken, refreshToken, scopes string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{
		"access_token":  accessToken,
		"token_type":    "bearer",
		"expires_in":    3600,
		"refresh_token": refreshToken,
		"scope":         scopes,
	})
}

func writeTokenError(w http.ResponseWriter, errCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": description,
	})
}

// ── Auth middleware ───────────────────────────────────────────────────────────

func requireBearerToken(cfg oauthConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="/.well-known/oauth-protected-resource"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		tok, err := jwt.Parse(tokenStr,
			func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return cfg.jwtSecret, nil
			},
			jwt.WithAudience(cfg.serverURL),
			jwt.WithExpirationRequired(),
		)
		if err != nil || !tok.Valid {
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="/.well-known/oauth-protected-resource"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ── Login page ────────────────────────────────────────────────────────────────

type loginPageData struct {
	ClientID      string
	RedirectURI   string
	State         string
	CodeChallenge string
	Scope         string
	Error         string
}

var loginTmpl = template.Must(template.New("login").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Triage — Authorize</title>
<style>
  body{font-family:system-ui,sans-serif;background:#f5f5f5;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}
  .card{background:#fff;border-radius:8px;padding:2rem;width:100%;max-width:360px;box-shadow:0 2px 8px rgba(0,0,0,.1)}
  h1{margin:0 0 .25rem;font-size:1.5rem}
  p{color:#666;margin:0 0 1.5rem;font-size:.9rem}
  label{display:block;font-size:.85rem;font-weight:500;margin-bottom:.25rem}
  input{width:100%;padding:.5rem .75rem;border:1px solid #ddd;border-radius:4px;font-size:1rem;box-sizing:border-box;margin-bottom:1rem}
  button{width:100%;padding:.6rem;background:#2563eb;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer}
  button:hover{background:#1d4ed8}
  .error{color:#dc2626;font-size:.85rem;margin-bottom:1rem}
</style>
</head>
<body>
<div class="card">
  <h1>Triage</h1>
  <p>Sign in to grant access</p>
  {{if .Error}}<div class="error">{{.Error}}</div>{{end}}
  <form method="POST" action="/oauth/login">
    <input type="hidden" name="client_id"       value="{{.ClientID}}">
    <input type="hidden" name="redirect_uri"    value="{{.RedirectURI}}">
    <input type="hidden" name="state"           value="{{.State}}">
    <input type="hidden" name="code_challenge"  value="{{.CodeChallenge}}">
    <input type="hidden" name="scope"           value="{{.Scope}}">
    <label for="u">Username</label>
    <input type="text"     id="u" name="username" required autofocus>
    <label for="p">Password</label>
    <input type="password" id="p" name="password" required>
    <button type="submit">Sign in</button>
  </form>
</div>
</body>
</html>`))

func renderLoginPage(w http.ResponseWriter, data loginPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	loginTmpl.Execute(w, data)
}
