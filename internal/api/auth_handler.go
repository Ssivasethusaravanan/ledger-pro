package api

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
	"ledger_pro/internal/db"
)

type LoginRequest struct {
	APIKey   string `json:"api_key"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Role string `json:"role"`
}

type MeResponse struct {
	Role string `json:"role"`
}

// LoginHandler verifies the provided credentials and sets a secure HttpOnly cookie.
// @Summary Login
// @Description Authenticates a user (via email/password or API key) and sets a secure session cookie
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ProblemDetail
// @Failure 401 {object} ProblemDetail
// @Router /v1/auth/login [post]
func (h *Handler) LoginHandler(keys map[string]APIKeyEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteProblem(w, r, BadRequest("invalid JSON body"))
			return
		}

		role := ""

		// 1. Verify Credentials
		if req.APIKey != "" {
			entry, found := keys[req.APIKey]
			if !found {
				WriteProblem(w, r, Unauthorized("invalid API key"))
				return
			}
			role = entry.Role
		} else if req.Email != "" && req.Password != "" {
			queries := db.New(h.pool)
			user, err := queries.GetUserByEmail(r.Context(), req.Email)
			if err != nil {
				WriteProblem(w, r, Unauthorized("invalid email or password"))
				return
			}
			
			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
				WriteProblem(w, r, Unauthorized("invalid email or password"))
				return
			}
			role = user.Role
		} else {
			WriteProblem(w, r, BadRequest("must provide api_key or email/password"))
			return
		}

		// 2. Generate Session Token in Redis
		token, err := h.auth.CreateSession(r.Context(), role)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "failed to create session", "error", err.Error())
			WriteProblem(w, r, InternalError("failed to create session"))
			return
		}

		// 3. Set HttpOnly Cookie
		cookie := &http.Cookie{
			Name:     "session_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   true, // must be served over HTTPS (Render enforces this)
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400, // 24 hours
		}
		http.SetCookie(w, cookie)

		writeJSON(w, http.StatusOK, LoginResponse{Role: role})
	}
}

// SignupHandler registers a new user.
// @Summary Signup
// @Description Registers a new user and sets a secure session cookie
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body SignupRequest true "Signup credentials"
// @Success 201 {object} LoginResponse
// @Failure 400 {object} ProblemDetail
// @Failure 409 {object} ProblemDetail
// @Router /v1/auth/signup [post]
func (h *Handler) SignupHandler(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteProblem(w, r, BadRequest("invalid JSON body"))
		return
	}

	if req.Email == "" || req.Password == "" {
		WriteProblem(w, r, BadRequest("email and password are required"))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		WriteProblem(w, r, InternalError("failed to process password"))
		return
	}

	queries := db.New(h.pool)
	user, err := queries.CreateUser(r.Context(), db.CreateUserParams{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "admin", // Defaulting to admin
	})
	if err != nil {
		// Basic check for unique constraint violation
		h.logger.ErrorContext(r.Context(), "failed to create user", "error", err.Error())
		WriteProblem(w, r, &ProblemDetail{
			Type:   "about:blank",
			Title:  "Conflict",
			Status: http.StatusConflict,
			Detail: "email already exists",
		})
		return
	}

	// Create session automatically after signup
	token, err := h.auth.CreateSession(r.Context(), user.Role)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "failed to create session", "error", err.Error())
		WriteProblem(w, r, InternalError("failed to create session"))
		return
	}

	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	}
	http.SetCookie(w, cookie)

	writeJSON(w, http.StatusCreated, LoginResponse{Role: user.Role})
}

// LogoutHandler clears the session cookie and removes it from Redis.
// @Summary Logout
// @Description Revokes the active session and clears the session cookie
// @Tags Authentication
// @Success 204 "No Content"
// @Router /v1/auth/logout [post]
func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		_ = h.auth.RevokeSession(r.Context(), cookie.Value)
	}

	// Clear cookie
	clearCookie := &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
	http.SetCookie(w, clearCookie)

	w.WriteHeader(http.StatusNoContent)
}

// MeHandler returns the currently authenticated user's role.
// @Summary Get Current User
// @Description Returns the role of the currently authenticated session
// @Tags Authentication
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MeResponse
// @Failure 401 {object} ProblemDetail
// @Router /v1/auth/me [get]
func (h *Handler) MeHandler(w http.ResponseWriter, r *http.Request) {
	role := GetAPIKeyRole(r.Context())
	if role == "" {
		WriteProblem(w, r, Unauthorized("not authenticated"))
		return
	}
	writeJSON(w, http.StatusOK, MeResponse{Role: role})
}
