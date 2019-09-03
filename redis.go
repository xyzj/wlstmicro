package wlstmicro

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/xyzj/gopsu"
)

var (
	// RedisClient redis客户端
	RedisClient *redis.Client
)

// NewRedisClient 新的redis client
func NewRedisClient() {
	if AppConf == nil {
		WriteLog("SYS", "Configuration files should be loaded first", 40)
		return
	}

	redisConf.addr = AppConf.GetItemDefault("redis_addr", "127.0.0.1:6379", "redis服务地址,ip:port格式")
	redisConf.pwd = gopsu.DecodeString(AppConf.GetItemDefault("redis_pwd", "WcELCNqP5dCpvMmMbKDdvgb", "redis连接密码"))
	redisConf.database, _ = strconv.Atoi(AppConf.GetItemDefault("redis_db", "0", "redis数据库名称"))
	redisConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("redis_enable", "true", "是否启用redis"))

	if !redisConf.enable {
		return
	}
	var err error

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisConf.addr,
		Password: redisConf.pwd,
		DB:       redisConf.database,
	})
	_, err = RedisClient.Ping().Result()
	if err != nil {
		WriteLog("REDIS", "Failed connect to server "+redisConf.addr+"|"+err.Error(), 40)
		return
	}
	activeRedis = true
	WriteLog("REDIS", "Success connect to server "+redisConf.addr, 90)
}

// WriteRedis 写redis
func WriteRedis(key string, value interface{}, expire time.Duration) {
	if RedisClient == nil {
		return
	}
	err := RedisClient.Set(key, value, expire).Err()
	if err != nil {
		WriteLog("REDIS", "Failed write redis data: "+err.Error(), 40)
	}
}

// EraseRedis 删redis
func EraseRedis(key ...string) {
	if RedisClient == nil {
		return
	}
	RedisClient.Del(key...)
}

// ReadRedis 读redis
func ReadRedis(key string) (string, error) {
	if RedisClient == nil {
		return "", fmt.Errorf("redis is not ready")
	}
	val := RedisClient.Get(key)
	if val.Err() != nil {
		WriteLog("REDIS", "Failed read redis data: "+key+"|"+val.Err().Error(), 40)
		return "", val.Err()
	}
	return val.Val(), nil
}

// RedisIsReady 返回redis可用状态
func RedisIsReady() bool {
	return RedisClient != nil
}
