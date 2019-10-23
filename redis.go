package wlstmicro

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/xyzj/gopsu"
)

var (
	// redisClient redis客户端
	redisClient *redis.Client
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

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisConf.addr,
		Password: redisConf.pwd,
		DB:       redisConf.database,
	})
	_, err = redisClient.Ping().Result()
	if err != nil {
		WriteLog("REDIS", "Failed connect to server "+redisConf.addr+"|"+err.Error(), 40)
		return
	}
	WriteLog("REDIS", "Success connect to server "+redisConf.addr, 90)
}

// AppendrootPathRedis 向redis的key追加头
func AppendrootPathRedis(key string) string {
	if !strings.HasPrefix(key, rootPathRedis()) {
		return rootPathRedis() + key
	}
	return key
}

// WriteRedis 写redis
func WriteRedis(key string, value interface{}, expire time.Duration) error {
	if redisClient == nil {
		return fmt.Errorf("redis is not ready")
	}
	err := redisClient.Set(AppendrootPathRedis(key), value, expire).Err()
	if err != nil {
		WriteLog("REDIS", "Failed write redis data: "+err.Error(), 40)
		return err
	}
	return nil
}

// EraseRedis 删redis
func EraseRedis(key ...string) {
	if redisClient == nil {
		return
	}
	keys := make([]string, len(key))
	for k, v := range key {
		keys[k] = AppendrootPathRedis(v)
	}
	redisClient.Del(keys...)
}

// EraseAllRedis 模糊删除
func EraseAllRedis(key string) {
	if redisClient == nil {
		return
	}
	val := redisClient.Keys(AppendrootPathRedis(key))
	if val.Err() != nil {
		return
	}
	redisClient.Del(val.Val()...)
}

// ReadRedis 读redis
func ReadRedis(key string) (string, error) {
	if redisClient == nil {
		return "", fmt.Errorf("redis is not ready")
	}
	val := redisClient.Get(AppendrootPathRedis(key))
	if val.Err() != nil {
		WriteLog("REDIS", "Failed read redis data: "+key+"|"+val.Err().Error(), 40)
		return "", val.Err()
	}
	return val.Val(), nil
}

// ReadAllRedis 模糊读redis
func ReadAllRedis(key string) ([]string, error) {
	if redisClient == nil {
		return []string{}, fmt.Errorf("redis is not ready")
	}
	val := redisClient.Keys(AppendrootPathRedis(key))
	if val.Err() != nil {
		WriteLog("REDIS", "Failed read redis data: "+key+"|"+val.Err().Error(), 40)
		return []string{}, val.Err()
	}
	var s = make([]string, 0)
	for _, v := range val.Val() {
		vv := redisClient.Get(v)
		if vv.Err() == nil {
			s = append(s, vv.Val())
		}
	}
	return s, nil
}

// RedisIsReady 返回redis可用状态
func RedisIsReady() bool {
	return redisClient != nil
}
