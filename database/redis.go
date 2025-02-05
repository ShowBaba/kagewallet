package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	ctx         = context.Background()
)

func SetRedisKey(key string, value string, expiration time.Duration) error {
	return RedisClient.Set(ctx, key, value, expiration).Err()
}

func GetRedisKey(key string) (string, error) {
	return RedisClient.Get(ctx, key).Result()
}

func DeleteRedisKey(key string) error {
	return RedisClient.Del(ctx, key).Err()
}

func DeleteRedisKeysByPattern(pattern string) error {
	keys, err := RedisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return RedisClient.Del(ctx, keys...).Err()
	}
	return nil
}

func Add(key string, value interface{}) error {
	return RedisClient.SAdd(ctx, key, value).Err()
}

func HSet(key, childKey string, data interface{}) error {
	return RedisClient.HSet(ctx, key, childKey, data).Err()
}

func HGetAll(key string) (map[string]string, error) {
	return RedisClient.HGetAll(ctx, key).Result()
}

func HGet(key, childKey string) (string, error) {
	value, err := RedisClient.HGet(ctx, key, childKey).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}

func RedisPublish(channel, key string) error {
	return RedisClient.Publish(ctx, channel, key).Err()
}

func RedisSubscribe(channel string) *redis.PubSub {
	return RedisClient.Subscribe(ctx, channel)
}
