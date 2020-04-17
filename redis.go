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
func NewRedisClient() bool {
	if AppConf == nil {
		WriteError("SYS", "Configuration files should be loaded first")
		return false
	}

	redisConf.addr = AppConf.GetItemDefault("redis_addr", "127.0.0.1:6379", "redis服务地址,ip:port格式")
	redisConf.pwd = gopsu.DecodeString(AppConf.GetItemDefault("redis_pwd", "WcELCNqP5dCpvMmMbKDdvgb", "redis连接密码"))
	redisConf.database, _ = strconv.Atoi(AppConf.GetItemDefault("redis_db", "0", "redis数据库名称"))
	redisConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("redis_enable", "true", "是否启用redis"))
	AppConf.Save()
	redisConf.show()
	if !redisConf.enable {
		return false
	}
	var err error

	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisConf.addr,
		Password: redisConf.pwd,
		DB:       redisConf.database,
	})
	_, err = redisClient.Ping().Result()
	if err != nil {
		WriteError("REDIS", "Failed connect to server "+redisConf.addr+"|"+err.Error())
		return false
	}
	WriteSystem("REDIS", "Success connect to server "+redisConf.addr)
	return true
}

// AppendRootPathRedis 向redis的key追加头
func AppendRootPathRedis(key string) string {
	if !strings.HasPrefix(key, rootPathRedis()) {
		return rootPathRedis() + key
	}
	return key
}

//ExpireRedis 更新redis有效期
func ExpireRedis(key string, expire time.Duration) error {
	if !redisConf.enable {
		return nil
	}
	if redisClient == nil {
		return fmt.Errorf("redis is not ready")
	}
	err := redisClient.Expire(AppendRootPathRedis(key), expire).Err()
	if err != nil {
		WriteError("REDIS", "Failed update redis expire: "+key+"|"+err.Error())
		return err
	}
	WriteInfo("REDIS", "Expire redis key: "+key)
	return nil
}

// WriteRedis 写redis
func WriteRedis(key string, value interface{}, expire time.Duration) error {
	if !redisConf.enable {
		return nil
	}
	if redisClient == nil {
		return fmt.Errorf("redis is not ready")
	}
	err := redisClient.Set(AppendRootPathRedis(key), value, expire).Err()
	if err != nil {
		WriteError("REDIS", "Failed write redis data: "+key+"|"+err.Error())
		return err
	}
	return nil
}

// EraseRedis 删redis
func EraseRedis(key ...string) {
	if !redisConf.enable || redisClient == nil {
		return
	}
	keys := make([]string, len(key))
	for k, v := range key {
		keys[k] = AppendRootPathRedis(v)
	}
	err := redisClient.Del(keys...).Err()
	if err != nil {
		WriteError("REDIS", fmt.Sprintf("Failed erase redis data: %+v|%s", keys, err.Error()))
	}
	WriteInfo("REDIS", fmt.Sprintf("Erase redis data:%+v", keys))
}

// EraseAllRedis 模糊删除
func EraseAllRedis(key string) {
	if !redisConf.enable || redisClient == nil {
		return
	}
	val := redisClient.Keys(AppendRootPathRedis(key))
	if val.Err() != nil {
		return
	}
	if len(val.Val()) > 0 {
		err := redisClient.Del(val.Val()...).Err()
		if err != nil {
			WriteError("REDIS", "Failed erase all redis data: "+key+"|"+err.Error())
		}
		WriteInfo("REDIS", fmt.Sprintf("Erase redis data:%s", AppendRootPathRedis(key)))
	}
}

// ReadRedis 读redis
func ReadRedis(key string) (string, error) {
	if redisClient == nil {
		return "", fmt.Errorf("redis is not ready")
	}
	key = AppendRootPathRedis(key)
	val := redisClient.Get(key)
	if val.Err() != nil {
		WriteError("REDIS", "Failed read redis data: "+key+"|"+val.Err().Error())
		return "", val.Err()
	}
	return val.Val(), nil
}

// ReadAllRedisKeys 模糊读取所有匹配的key
func ReadAllRedisKeys(key string) *redis.StringSliceCmd {
	if !redisConf.enable {
		return &redis.StringSliceCmd{}
	}
	return redisClient.Keys(AppendRootPathRedis(key))
}

// ReadAllRedis 模糊读redis
func ReadAllRedis(key string) ([]string, error) {
	if redisClient == nil {
		return []string{}, fmt.Errorf("redis is not ready")
	}
	key = AppendRootPathRedis(key)
	val := redisClient.Keys(key)
	if val.Err() != nil {
		WriteError("REDIS", "Failed read redis data: "+key+"|"+val.Err().Error())
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

// ViewRedisConfig 查看redis配置,返回json字符串
func ViewRedisConfig() string {
	return redisConf.forshow
}

// ExpireUserToken 更新token有效期
func ExpireUserToken(token string) {
	// 更新redis的对应键值的有效期
	go ExpireRedis("usermanager/legal/"+MD5Worker.Hash([]byte(token)), time.Minute*30)
}
