package link

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type repositoryStub struct {
	values map[string]Link
	err    error
}

func (r *repositoryStub) Create(_ context.Context, value Link) error {
	if r.err != nil {
		return r.err
	}
	if _, exists := r.values[value.Code]; exists {
		return ErrConflict
	}
	r.values[value.Code] = value
	return nil
}

func (r *repositoryStub) Get(_ context.Context, code string) (Link, error) {
	value, exists := r.values[code]
	if !exists {
		return Link{}, ErrNotFound
	}
	return value, nil
}

type cacheStub struct {
	values map[string]Link
	err    error
}

func (c *cacheStub) Get(_ context.Context, code string) (Link, error) {
	if c.err != nil {
		return Link{}, c.err
	}
	value, exists := c.values[code]
	if !exists {
		return Link{}, ErrNotFound
	}
	return value, nil
}

func (c *cacheStub) Set(_ context.Context, value Link, _ time.Duration) error {
	c.values[value.Code] = value
	return nil
}

func (c *cacheStub) Delete(_ context.Context, code string) error {
	delete(c.values, code)
	return nil
}

func TestCreateAndResolve(t *testing.T) {
	repository := &repositoryStub{values: make(map[string]Link)}
	cache := &cacheStub{values: make(map[string]Link)}
	service := NewService(repository, cache, time.Hour, 0)
	service.now = func() time.Time { return time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC) }
	service.newCode = func() (string, error) { return "Ab12Cd34", nil }

	created, err := service.Create(context.Background(), "https://example.com/page?q=1", nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Code != "Ab12Cd34" {
		t.Fatalf("Create() code = %q", created.Code)
	}

	resolved, err := service.Resolve(context.Background(), created.Code)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.OriginalURL != created.OriginalURL {
		t.Fatalf("Resolve() URL = %q", resolved.OriginalURL)
	}
}

func TestResolveFallsBackToRepository(t *testing.T) {
	value := Link{Code: "Ab12Cd34", OriginalURL: "https://example.com", CreatedAt: time.Now().UTC()}
	repository := &repositoryStub{values: map[string]Link{value.Code: value}}
	cache := &cacheStub{values: make(map[string]Link), err: errors.New("redis unavailable")}
	service := NewService(repository, cache, time.Hour, 0)

	resolved, err := service.Resolve(context.Background(), value.Code)
	if err != nil || resolved.OriginalURL != value.OriginalURL {
		t.Fatalf("Resolve() = %#v, %v", resolved, err)
	}
}

func TestResolveExpired(t *testing.T) {
	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	expires := now.Add(-time.Minute)
	value := Link{Code: "Ab12Cd34", OriginalURL: "https://example.com", ExpiresAt: &expires}
	repository := &repositoryStub{values: map[string]Link{value.Code: value}}
	cache := &cacheStub{values: map[string]Link{value.Code: value}}
	service := NewService(repository, cache, time.Hour, 0)
	service.now = func() time.Time { return now }

	_, err := service.Resolve(context.Background(), value.Code)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Resolve() error = %v, want ErrNotFound", err)
	}
}

func TestCreateRejectsInvalidURL(t *testing.T) {
	service := NewService(&repositoryStub{values: make(map[string]Link)}, &cacheStub{values: make(map[string]Link)}, time.Hour, 0)
	for _, raw := range []string{"", "example.com", "ftp://example.com", "https://user:pass@example.com"} {
		if _, err := service.Create(context.Background(), raw, nil); err == nil {
			t.Errorf("Create(%q) expected error", raw)
		}
	}
}

func TestCreateAcceptsLongURL(t *testing.T) {
	repository := &repositoryStub{values: make(map[string]Link)}
	service := NewService(repository, &cacheStub{values: make(map[string]Link)}, time.Hour, 0)
	service.newCode = func() (string, error) { return "Ab12Cd34", nil }
	raw := "https://example.com/?data=" + strings.Repeat("a", maxOriginalURLLength-len("https://example.com/?data="))

	created, err := service.Create(context.Background(), raw, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.OriginalURL != raw {
		t.Errorf("Create() URL length = %d, want %d", len(created.OriginalURL), len(raw))
	}
}

func TestCreateRejectsTooLongURL(t *testing.T) {
	service := NewService(&repositoryStub{values: make(map[string]Link)}, &cacheStub{values: make(map[string]Link)}, time.Hour, 0)
	raw := "https://example.com/?data=" + strings.Repeat("a", maxOriginalURLLength-len("https://example.com/?data=")+1)

	_, err := service.Create(context.Background(), raw, nil)
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("Create() error = %v, want ErrInvalidURL", err)
	}
}
