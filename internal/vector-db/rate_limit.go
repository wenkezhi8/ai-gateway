package vectordb

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type VectorSearchRateLimiter struct {
	maxRequests int
	window      time.Duration
	redisClient *redis.Client

	mu       sync.Mutex
	counters map[string]*rateCounter
}

type rateCounter struct {
	count   int
	resetAt time.Time
}

func NewVectorSearchRateLimiter(maxRequests int, window time.Duration) *VectorSearchRateLimiter {
	if maxRequests <= 0 {
		maxRequests = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	client := newVectorRedisClient()
	return &VectorSearchRateLimiter{
		maxRequests: maxRequests,
		window:      window,
		redisClient: client,
		counters:    make(map[string]*rateCounter),
	}
}

func (l *VectorSearchRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil {
			c.Next()
			return
		}

		key := l.requestKey(c)
		allowed, retryAfter := l.allowNow(key)
		if allowed {
			c.Next()
			return
		}

		c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
		c.JSON(http.StatusTooManyRequests, gin.H{"success": false, "error": "rate limit exceeded"})
		c.Abort()
	}
}

func (l *VectorSearchRateLimiter) requestKey(c *gin.Context) string {
	apiKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
	if apiKey == "" {
		authorization := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
			apiKey = strings.TrimSpace(authorization[len("Bearer "):])
		}
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(c.ClientIP())
	}
	collection := strings.TrimSpace(c.Param("name"))
	if collection == "" {
		collection = "unknown"
	}
	return apiKey + ":" + collection
}

func (l *VectorSearchRateLimiter) allowNow(key string) (allowed bool, retryAfter int64) {
	if l.redisClient != nil {
		allowed, retryAfter, err := l.allowNowWithRedis(key)
		if err == nil {
			return allowed, retryAfter
		}
	}

	now := time.Now().UTC()

	l.mu.Lock()
	defer l.mu.Unlock()

	counter, ok := l.counters[key]
	if !ok || now.After(counter.resetAt) {
		l.counters[key] = &rateCounter{count: 1, resetAt: now.Add(l.window)}
		return true, 0
	}
	if counter.count >= l.maxRequests {
		retryAfter := int64(time.Until(counter.resetAt).Seconds())
		if retryAfter <= 0 {
			retryAfter = 1
		}
		return false, retryAfter
	}

	counter.count++
	return true, 0
}

func (l *VectorSearchRateLimiter) allowNowWithRedis(key string) (allowed bool, retryAfter int64, err error) {
	ctx := context.Background()
	redisKey := "vector:rate:" + key
	now := time.Now().UTC().UnixNano()
	windowNanos := l.window.Nanoseconds()

	script := `
local current = redis.call('GET', KEYS[1])
if current == false then
  redis.call('SET', KEYS[1], 1, 'PX', ARGV[1])
  return {1, 1, ARGV[1]}
end

local value = tonumber(current)
if value >= tonumber(ARGV[2]) then
  local ttl = redis.call('PTTL', KEYS[1])
  if ttl < 0 then ttl = ARGV[1] end
  return {0, value, ttl}
end

local newval = redis.call('INCR', KEYS[1])
local ttl = redis.call('PTTL', KEYS[1])
if ttl < 0 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
  ttl = ARGV[1]
end
return {1, newval, ttl}
`

	result, err := l.redisClient.Eval(ctx, script, []string{redisKey}, windowNanos/1_000_000, l.maxRequests, now).Result()
	if err != nil {
		return false, 0, err
	}

	arr, ok := result.([]interface{})
	if !ok || len(arr) < 3 {
		return false, 0, fmt.Errorf("unexpected redis eval result")
	}
	allowedFlag, ok := arr[0].(int64)
	if !ok {
		return false, 0, fmt.Errorf("unexpected allowed flag type")
	}
	if allowedFlag == 1 {
		return true, 0, nil
	}
	ttlMillis, ok := arr[2].(int64)
	if !ok {
		return false, 1, nil
	}
	retryAfter = ttlMillis / 1000
	if retryAfter <= 0 {
		retryAfter = 1
	}
	return false, retryAfter, nil
}

func newVectorRedisClient() *redis.Client {
	addr := strings.TrimSpace(os.Getenv("AI_GATEWAY_REDIS_ADDR"))
	if addr == "" {
		return nil
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil
	}
	return client
}
