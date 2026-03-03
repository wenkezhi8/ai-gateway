package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCapabilityRedisExecutor struct {
	responses map[string]error
}

func (f *fakeCapabilityRedisExecutor) Do(ctx context.Context, args ...interface{}) *redis.Cmd {
	command := ""
	if len(args) > 0 {
		if cmd, ok := args[0].(string); ok {
			command = cmd
		}
	}
	err := f.responses[command]
	cmd := redis.NewCmd(ctx, args...)
	if err != nil {
		cmd.SetErr(err)
		return cmd
	}
	cmd.SetVal("OK")
	return cmd
}

func TestEnsureRedisStackCapabilitiesWithExecutor_ShouldFailOnMissingRediSearch(t *testing.T) {
	executor := &fakeCapabilityRedisExecutor{responses: map[string]error{
		"FT._LIST": errors.New("ERR unknown command 'FT._LIST'"),
	}}

	err := ensureRedisStackCapabilitiesWithExecutor(context.Background(), executor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RediSearch")
}

func TestEnsureRedisStackCapabilitiesWithExecutor_ShouldFailOnMissingRedisJSON(t *testing.T) {
	executor := &fakeCapabilityRedisExecutor{responses: map[string]error{
		"FT._LIST": nil,
		"JSON.GET": errors.New("ERR unknown command 'JSON.GET'"),
	}}

	err := ensureRedisStackCapabilitiesWithExecutor(context.Background(), executor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RedisJSON")
}

func TestEnsureRedisStackCapabilitiesWithExecutor_ShouldAcceptRedisNilForJsonGet(t *testing.T) {
	executor := &fakeCapabilityRedisExecutor{responses: map[string]error{
		"FT._LIST": nil,
		"JSON.GET": redis.Nil,
	}}

	err := ensureRedisStackCapabilitiesWithExecutor(context.Background(), executor)
	require.NoError(t, err)
}
