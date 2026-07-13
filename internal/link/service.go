package link

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrNotFound      = errors.New("link not found")
	ErrConflict      = errors.New("link code conflict")
	ErrInvalidURL    = errors.New("invalid URL")
	ErrInvalidExpiry = errors.New("invalid expiry")
)

const maxOriginalURLLength = 65535

type Repository interface {
	Create(context.Context, Link) error
	Get(context.Context, string) (Link, error)
}

type Cache interface {
	Get(context.Context, string) (Link, error)
	Set(context.Context, Link, time.Duration) error
	Delete(context.Context, string) error
}

type Service struct {
	repository Repository
	cache      Cache
	cacheTTL   time.Duration
	defaultTTL time.Duration
	now        func() time.Time
	newCode    func() (string, error)
}

func NewService(repository Repository, cache Cache, cacheTTL, defaultTTL time.Duration) *Service {
	return &Service{
		repository: repository,
		cache:      cache,
		cacheTTL:   cacheTTL,
		defaultTTL: defaultTTL,
		now:        time.Now,
		newCode:    generateCode,
	}
}

func (s *Service) Create(ctx context.Context, originalURL string, expiresAt *time.Time) (Link, error) {
	normalized, err := validateURL(originalURL)
	if err != nil {
		return Link{}, err
	}

	now := s.now().UTC()
	if expiresAt == nil && s.defaultTTL > 0 {
		expires := now.Add(s.defaultTTL)
		expiresAt = &expires
	}
	if expiresAt != nil {
		expires := expiresAt.UTC()
		if !expires.After(now) {
			return Link{}, fmt.Errorf("%w: expires_at must be in the future", ErrInvalidExpiry)
		}
		expiresAt = &expires
	}

	for range 5 {
		code, codeErr := s.newCode()
		if codeErr != nil {
			return Link{}, fmt.Errorf("generate code: %w", codeErr)
		}
		created := Link{Code: code, OriginalURL: normalized, CreatedAt: now, ExpiresAt: expiresAt}
		err = s.repository.Create(ctx, created)
		if err == nil {
			_ = s.cache.Set(ctx, created, s.ttl(created, now))
			return created, nil
		}
		if !errors.Is(err, ErrConflict) {
			return Link{}, err
		}
	}
	return Link{}, fmt.Errorf("generate unique code: %w", ErrConflict)
}

func (s *Service) Resolve(ctx context.Context, code string) (Link, error) {
	if !validCode(code) {
		return Link{}, ErrNotFound
	}
	now := s.now().UTC()
	if cached, err := s.cache.Get(ctx, code); err == nil {
		if cached.Expired(now) {
			_ = s.cache.Delete(ctx, code)
			return Link{}, ErrNotFound
		}
		return cached, nil
	}

	stored, err := s.repository.Get(ctx, code)
	if err != nil {
		return Link{}, err
	}
	if stored.Expired(now) {
		_ = s.cache.Delete(ctx, code)
		return Link{}, ErrNotFound
	}
	_ = s.cache.Set(ctx, stored, s.ttl(stored, now))
	return stored, nil
}

func (s *Service) ttl(value Link, now time.Time) time.Duration {
	ttl := s.cacheTTL
	if value.ExpiresAt != nil {
		untilExpiry := value.ExpiresAt.Sub(now)
		if ttl <= 0 || untilExpiry < ttl {
			ttl = untilExpiry
		}
	}
	return ttl
}

func validateURL(raw string) (string, error) {
	length := utf8.RuneCountInString(raw)
	if length == 0 || length > maxOriginalURLLength {
		return "", fmt.Errorf("%w: original_url length must be between 1 and %d characters", ErrInvalidURL, maxOriginalURLLength)
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("%w: original_url must be an absolute http or https URL", ErrInvalidURL)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("%w: original_url must not contain credentials", ErrInvalidURL)
	}
	return parsed.String(), nil
}

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func generateCode() (string, error) {
	code := make([]byte, 8)
	buffer := make([]byte, 16)
	for written := 0; written < len(code); {
		if _, err := rand.Read(buffer); err != nil {
			return "", err
		}
		for _, value := range buffer {
			// 248 is the largest multiple of 62 below 256. Discarding the
			// remaining values avoids modulo bias.
			if value >= 248 {
				continue
			}
			code[written] = alphabet[int(value)%len(alphabet)]
			written++
			if written == len(code) {
				break
			}
		}
	}
	return string(code), nil
}

func validCode(code string) bool {
	if len(code) != 8 {
		return false
	}
	for _, char := range code {
		if !strings.ContainsRune(alphabet, char) {
			return false
		}
	}
	return true
}
