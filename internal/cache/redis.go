package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vladislav/short/internal/link"
)

type Redis struct{ client *redis.Client }

func Open(ctx context.Context, addr, password string, db int) (*Redis, error) {
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &Redis{client: client}, nil
}

func (r *Redis) Close() error                   { return r.client.Close() }
func (r *Redis) Ping(ctx context.Context) error { return r.client.Ping(ctx).Err() }

func (r *Redis) Get(ctx context.Context, code string) (link.Link, error) {
	payload, err := r.client.Get(ctx, key(code)).Bytes()
	if errors.Is(err, redis.Nil) {
		return link.Link{}, link.ErrNotFound
	}
	if err != nil {
		return link.Link{}, fmt.Errorf("get cached link: %w", err)
	}
	var value link.Link
	if err = json.Unmarshal(payload, &value); err != nil {
		return link.Link{}, fmt.Errorf("decode cached link: %w", err)
	}
	return value, nil
}

func (r *Redis) Set(ctx context.Context, value link.Link, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode cached link: %w", err)
	}
	if err = r.client.Set(ctx, key(value.Code), payload, ttl).Err(); err != nil {
		return fmt.Errorf("cache link: %w", err)
	}
	return nil
}

func (r *Redis) Delete(ctx context.Context, code string) error {
	return r.client.Del(ctx, key(code)).Err()
}

func key(code string) string { return "short:link:" + code }
