package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

type CacheMiddleware struct {
	cache *QueryCache
	ttl   time.Duration
}

func NewCacheMiddleware(ttl time.Duration) *CacheMiddleware {
	return &CacheMiddleware{
		cache: NewQueryCache(),
		ttl:   ttl,
	}
}

func (m *CacheMiddleware) CacheQuery(ctx context.Context, next graphql.Resolver) (interface{}, error) {
	// Get operation context
	oc := graphql.GetOperationContext(ctx)
	fc := graphql.GetFieldContext(ctx)

	// Only cache queries
	if oc.Operation.Operation != "query" {
		return next(ctx)
	}

	// Generate cache key from query and variables
	rawQuery := oc.RawQuery
	variables, _ := json.Marshal(oc.Variables)
	key := generateCacheKey(rawQuery, string(variables), fc.Field.Name)

	// Check cache
	if cached, exists := m.cache.Get(key); exists {
		return cached, nil
	}

	// Execute query
	result, err := next(ctx)
	if err != nil {
		return nil, err
	}

	// Cache result
	m.cache.Set(key, result, m.ttl)
	return result, nil
}

func generateCacheKey(query, variables, fieldName string) string {
	hash := sha256.New()
	hash.Write([]byte(query + variables + fieldName))
	return hex.EncodeToString(hash.Sum(nil))
}
