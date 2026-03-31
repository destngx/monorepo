package domain

import "context"

type bypassCacheContextKey struct{}

func WithBypassCache(ctx context.Context, bypass bool) context.Context {
	return context.WithValue(ctx, bypassCacheContextKey{}, bypass)
}

func ShouldBypassCache(ctx context.Context) bool {
	value := ctx.Value(bypassCacheContextKey{})
	bypass, ok := value.(bool)
	return ok && bypass
}
