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
func WriteRedis(key string, value interface{}, expire time.Duration) error {
	if RedisClient == nil {
		return fmt.Errorf("redis is not ready")
	}
	key = "/" + RootPath + key
	err := RedisClient.Set(key, value, expire).Err()
	if err != nil {
		WriteLog("REDIS", "Failed write redis data: "+err.Error(), 40)
		return err
	}
	return nil
}

// EraseRedis 删redis
func EraseRedis(key ...string) {
	if RedisClient == nil {
		return
	}
	keys := make([]string, len(key))
	for k, v := range key {
		keys[k] = "/" + RootPath + v
	}
	RedisClient.Del(keys...)
}

// EraseAllRedis 模糊删除
func EraseAllRedis(key string) {
	if RedisClient == nil {
		return
	}
	key = "/" + RootPath + key
	val := RedisClient.Keys(key)
	if val.Err() != nil {
		return
	}
	RedisClient.Del(val.Val()...)
}

// ReadRedis 读redis
func ReadRedis(key string) (string, error) {
	if RedisClient == nil {
		return "", fmt.Errorf("redis is not ready")
	}
	key = "/" + RootPath + key
	val := RedisClient.Get(key)
	if val.Err() != nil {
		WriteLog("REDIS", "Failed read redis data: "+key+"|"+val.Err().Error(), 40)
		return "", val.Err()
	}
	return val.Val(), nil
}

// ReadAllRedis 模糊读redis
func ReadAllRedis(key string) ([]string, error) {
	if RedisClient == nil {
		return []string{}, fmt.Errorf("redis is not ready")
	}
	key = "/" + RootPath + key
	val := RedisClient.Keys(key)
	if val.Err() != nil {
		WriteLog("REDIS", "Failed read redis data: "+key+"|"+val.Err().Error(), 40)
		return []string{}, val.Err()
	}
	var s = make([]string, 0)
	for _, v := range val.Val() {
		vv := RedisClient.Get(v)
		if vv.Err() == nil {
			s = append(s, vv.Val())
		}
	}
	return s, nil
}

// RedisIsReady 返回redis可用状态
func RedisIsReady() bool {
	return RedisClient != nil
}
