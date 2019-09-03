package wlstmicro

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/xyzj/gopsu"
)

var (
	redisClient *redis.Client
)

// NewRedisClient 新的redis client
func NewRedisClient() {
	if appConf == nil {
		writeLog("SYS", "Configuration files should be loaded first", 40)
		return
	}

	redisConf.addr = appConf.GetItemDefault("redis_addr", "127.0.0.1:6379", "redis服务地址,ip:port格式")
	redisConf.pwd = gopsu.DecodeString(appConf.GetItemDefault("redis_pwd", "WcELCNqP5dCpvMmMbKDdvgb", "redis连接密码"))
	redisConf.database, _ = strconv.Atoi(appConf.GetItemDefault("redis_db", "0", "redis数据库名称"))
	if !standAloneMode {
		redisConf.enable, _ = strconv.ParseBool(appConf.GetItemDefault("redis_enable", "true", "是否启用redis"))
	}

	if !redisConf.enable {
		return
	}
	var err error

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisConf.addr,
		Password: redisConf.pwd,
		DB:       redisConf.database,
	})
	_, err = redisClient.Ping().Result()
	if err != nil {
		writeLog("REDIS", "Failed connect to server "+redisConf.addr+"|"+err.Error(), 40)
		return
	}
	activeRedis = true
	writeLog("REDIS", "Success connect to server "+redisConf.addr, 90)
}

func writeRedis(key string, value interface{}, expire time.Duration) {
	if redisClient == nil {
		return
	}
	err := redisClient.Set(key, value, expire).Err()
	if err != nil {
		writeLog("REDIS", "Failed write redis data: "+err.Error(), 40)
	}
}

func eraseRedis(key ...string) {
	if redisClient == nil {
		return
	}
	redisClient.Del(key...)
}

func readRedis(key string) (string, error) {
	if redisClient == nil {
		return "", fmt.Errorf("redis is not ready")
	}
	val := redisClient.Get(key)
	if val.Err() != nil {
		writeLog("REDIS", "Failed read redis data: "+key+"|"+val.Err().Error(), 40)
		return "", val.Err()
	}
	return val.Val(), nil
}

// RedisIsReady 返回redis可用状态
func RedisIsReady() bool {
	return redisClient != nil
}
