package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidSession = errors.New("invalid or expired session token")
)

const sessionKeyPrefix = "session:"

// SessionData holds the authenticated user's session state
type SessionData struct {
	Role     string    `json:"role"`
	TenantID uuid.UUID `json:"tenant_id"`
}

// AuthService handles creating and validating browser session tokens using Redis.
type AuthService struct {
	rdb        *redis.Client
	sessionTTL time.Duration
}

// NewAuthService constructs a new AuthService.
func NewAuthService(rdb *redis.Client, ttl time.Duration) *AuthService {
	return &AuthService{
		rdb:        rdb,
		sessionTTL: ttl,
	}
}

// CreateSession generates a secure random token and stores it in Redis with the associated session data.
func (s *AuthService) CreateSession(ctx context.Context, role string, tenantID uuid.UUID) (string, error) {
	// Generate a 32-byte secure random token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(b)

	sessionData := SessionData{
		Role:     role,
		TenantID: tenantID,
	}

	data, err := json.Marshal(sessionData)
	if err != nil {
		return "", fmt.Errorf("marshal session: %w", err)
	}

	// Store in Redis
	key := sessionKeyPrefix + token
	if err := s.rdb.Set(ctx, key, data, s.sessionTTL).Err(); err != nil {
		return "", fmt.Errorf("store session in redis: %w", err)
	}

	return token, nil
}

// ValidateSession checks Redis for the token and returns the session data.
func (s *AuthService) ValidateSession(ctx context.Context, token string) (*SessionData, error) {
	key := sessionKeyPrefix + token

	// Pipeline to GET and EXPIRE atomically
	pipe := s.rdb.Pipeline()
	getCmd := pipe.Get(ctx, key)
	pipe.Expire(ctx, key, s.sessionTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrInvalidSession
		}
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	data, err := getCmd.Result()
	if err != nil {
		return nil, fmt.Errorf("get result: %w", err)
	}

	var sessionData SessionData
	if err := json.Unmarshal([]byte(data), &sessionData); err != nil {
		// Fallback for old sessions that were just plain text roles
		if data == "admin" || data == "write" || data == "read" {
			uid, _ := uuid.Parse("00000000-0000-0000-0000-000000000000")
			return &SessionData{Role: data, TenantID: uid}, nil
		}
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return &sessionData, nil
}

// RevokeSession explicitly deletes the session token from Redis (Logout).
func (s *AuthService) RevokeSession(ctx context.Context, token string) error {
	key := sessionKeyPrefix + token
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
