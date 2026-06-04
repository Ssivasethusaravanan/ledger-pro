package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidSession = errors.New("invalid or expired session token")
)

const sessionKeyPrefix = "session:"

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

// CreateSession generates a secure random token and stores it in Redis with the associated role.
func (s *AuthService) CreateSession(ctx context.Context, role string) (string, error) {
	// Generate a 32-byte secure random token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(b)

	// Store in Redis
	key := sessionKeyPrefix + token
	if err := s.rdb.Set(ctx, key, role, s.sessionTTL).Err(); err != nil {
		return "", fmt.Errorf("store session in redis: %w", err)
	}

	return token, nil
}

// ValidateSession checks Redis for the token and returns the associated role.
// It also refreshes the TTL of the session to keep it alive.
func (s *AuthService) ValidateSession(ctx context.Context, token string) (string, error) {
	key := sessionKeyPrefix + token

	// Pipeline to GET and EXPIRE atomically
	pipe := s.rdb.Pipeline()
	getCmd := pipe.Get(ctx, key)
	pipe.Expire(ctx, key, s.sessionTTL)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrInvalidSession
		}
		return "", fmt.Errorf("pipeline exec: %w", err)
	}

	role, err := getCmd.Result()
	if err != nil {
		return "", fmt.Errorf("get result: %w", err)
	}

	return role, nil
}

// RevokeSession explicitly deletes the session token from Redis (Logout).
func (s *AuthService) RevokeSession(ctx context.Context, token string) error {
	key := sessionKeyPrefix + token
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
