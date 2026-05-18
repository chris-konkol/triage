package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	authjwt "github.com/chris-konkol/triage/internal/auth"
)

type authHandlers struct {
	db        *pgxpool.Pool
	jwtSecret string
}

func (h *authHandlers) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var userID string
	err = h.db.QueryRow(r.Context(), `
		INSERT INTO users (username, email, password_hash, role)
		VALUES ($1, $2, $3, 'submitter')
		RETURNING id
	`, req.Username, req.Email, string(hash)).Scan(&userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "username or email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	token, err := authjwt.Generate(userID, req.Username, "submitter", h.jwtSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"token":    token,
		"userId":   userID,
		"username": req.Username,
		"role":     "submitter",
	})
}

func (h *authHandlers) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var userID, passwordHash, role string
	err := h.db.QueryRow(r.Context(), `
		SELECT id, password_hash, role FROM users WHERE username = $1
	`, req.Username).Scan(&userID, &passwordHash, &role)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := authjwt.Generate(userID, req.Username, role, h.jwtSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":    token,
		"userId":   userID,
		"username": req.Username,
		"role":     role,
	})
}
