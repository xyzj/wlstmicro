package wmv2

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
)

var (
	redisCtxTimeo = 3 * time.Second
)

// redis配置
type redisConfigure struct {
	forshow string
	// redis服务地址
	addr string
	// 访问密码
	pwd string
	// 数据库
	database int
	// 是否启用redis
	enable bool
	// client
	client *redis.Client
}

func (conf *redisConfigure) show(rootPath string) string {
	conf.forshow, _ = sjson.Set("", "addr", conf.addr)
	conf.forshow, _ = sjson.Set(conf.forshow, "pwd", CWorker.Encrypt(conf.pwd))
	conf.forshow, _ = sjson.Set(conf.forshow, "dbname", conf.database)
	conf.forshow, _ = sjson.Set(conf.forshow, "root_path", rootPath)
	return conf.forshow
}

// NewRedisClient 新的redis client
func (fw *WMFrameWorkV2) newRedisClient() bool {
	fw.redisCtl.addr = fw.wmConf.GetItemDefault("redis_addr", "127.0.0.1:6379", "redis服务地址,ip:port格式")
	fw.redisCtl.pwd = gopsu.DecodeString(fw.wmConf.GetItemDefault("redis_pwd", "WcELCNqP5dCpvMmMbKDdvgb", "redis连接密码"))
	fw.redisCtl.database, _ = strconv.Atoi(fw.wmConf.GetItemDefault("redis_db", "0", "redis数据库名称"))
	fw.redisCtl.enable, _ = strconv.ParseBool(fw.wmConf.GetItemDefault("redis_enable", "true", "是否启用redis"))
	fw.wmConf.Save()
	fw.redisCtl.show(fw.rootPath)
	if !fw.redisCtl.enable {
		return false
	}
	var err error

	fw.redisCtl.client = redis.NewClient(&redis.Options{
		Addr:     fw.redisCtl.addr,
		Password: fw.redisCtl.pwd,
		DB:       fw.redisCtl.database,
	})
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	_, err = fw.redisCtl.client.Ping(ctx).Result()
	if err != nil {
		fw.redisCtl.enable = false
		fw.WriteError("REDIS", "Failed connect to server "+fw.redisCtl.addr+"|"+err.Error())
		return false
	}
	fw.WriteSystem("REDIS", "Success connect to server "+fw.redisCtl.addr)
	return true
}

// AppendRootPathRedis 向redis的key追加头
func (fw *WMFrameWorkV2) AppendRootPathRedis(key string) string {
	if !strings.HasPrefix(key, fw.rootPathRedis) {
		return fw.rootPathRedis + key
	}
	return key
}

//ExpireRedis 更新redis有效期
func (fw *WMFrameWorkV2) ExpireRedis(key string, expire time.Duration) error {
	if !fw.redisCtl.enable {
		return fmt.Errorf("redis is not ready")
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	err := fw.redisCtl.client.Expire(ctx, fw.AppendRootPathRedis(key), expire).Err()
	if err != nil {
		fw.WriteError("REDIS", "Failed update redis expire: "+key+"|"+err.Error())
		return err
	}
	fw.WriteDebug("REDIS", "Expire redis key: "+key)
	return nil
}

// WriteRedis 写redis
func (fw *WMFrameWorkV2) WriteRedis(key string, value interface{}, expire time.Duration) error {
	if !fw.redisCtl.enable {
		return fmt.Errorf("redis is not ready")
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	err := fw.redisCtl.client.Set(ctx, fw.AppendRootPathRedis(key), value, expire).Err()
	if err != nil {
		fw.WriteError("REDIS", "Failed write redis data: "+key+"|"+err.Error())
		return err
	}
	return nil
}

// EraseRedis 删redis
func (fw *WMFrameWorkV2) EraseRedis(key ...string) error {
	if !fw.redisCtl.enable {
		return fmt.Errorf("redis is not ready")
	}
	keys := make([]string, len(key))
	for k, v := range key {
		keys[k] = fw.AppendRootPathRedis(v)
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	err := fw.redisCtl.client.Del(ctx, keys...).Err()
	if err != nil {
		fw.WriteError("REDIS", fmt.Sprintf("Failed erase redis data: %+v|%s", keys, err.Error()))
		return err
	}
	fw.WriteInfo("REDIS", fmt.Sprintf("Erase redis data:%+v", keys))
	return nil
}

// EraseAllRedis 模糊删除
func (fw *WMFrameWorkV2) EraseAllRedis(key string) error {
	if !fw.redisCtl.enable {
		return fmt.Errorf("redis is not ready")
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	val := fw.redisCtl.client.Keys(ctx, fw.AppendRootPathRedis(key))
	if val.Err() != nil {
		return val.Err()
	}
	if len(val.Val()) > 0 {
		ctx2, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
		defer cancel()
		err := fw.redisCtl.client.Del(ctx2, val.Val()...).Err()
		if err != nil {
			fw.WriteError("REDIS", "Failed erase all redis data: "+key+"|"+err.Error())
			return err
		}
		fw.WriteInfo("REDIS", fmt.Sprintf("Erase redis data:%s", fw.AppendRootPathRedis(key)))
	}
	return nil
}

// ReadRedis 读redis
func (fw *WMFrameWorkV2) ReadRedis(key string) (string, error) {
	if !fw.redisCtl.enable {
		return "", fmt.Errorf("redis is not ready")
	}
	key = fw.AppendRootPathRedis(key)
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	val := fw.redisCtl.client.Get(ctx, key)
	if val.Err() != nil {
		fw.WriteError("REDIS", "Failed read redis data: "+key+"|"+val.Err().Error())
		return "", val.Err()
	}
	return val.Val(), nil
}

// ReadHashRedis 读取所有hash数据
func (fw *WMFrameWorkV2) ReadHashRedis(key, field string) (string, error) {
	if !fw.redisCtl.enable {
		return "", fmt.Errorf("redis is not ready")
	}
	key = fw.AppendRootPathRedis(key)
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	val := fw.redisCtl.client.HGet(ctx, key, field)
	if val.Err() != nil {
		fw.WriteError("REDIS", "Failed read redis hash data: "+key+"|"+val.Err().Error())
		return "", val.Err()
	}
	return val.Val(), nil
}

// ReadHashAllRedis 读取所有hash数据
func (fw *WMFrameWorkV2) ReadHashAllRedis(key string) (map[string]string, error) {
	if !fw.redisCtl.enable {
		return nil, fmt.Errorf("redis is not ready")
	}
	key = fw.AppendRootPathRedis(key)
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	val := fw.redisCtl.client.HGetAll(ctx, key)
	if val.Err() != nil {
		fw.WriteError("REDIS", "Failed read redis hash data: "+key+"|"+val.Err().Error())
		return nil, val.Err()
	}
	return val.Val(), nil
}

// WriteHashFieldRedis 修改或添加redis hashmap中的值
func (fw *WMFrameWorkV2) WriteHashFieldRedis(key, field string, value interface{}) error {
	if !fw.redisCtl.enable {
		return fmt.Errorf("redis is not ready")
	}
	key = fw.AppendRootPathRedis(key)
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	val := fw.redisCtl.client.HSetNX(ctx, key, field, value)
	if val.Err() != nil {
		fw.WriteError("REDIS", "Failed write redis hash data: "+key+"|"+val.Err().Error())
		return val.Err()
	}
	return nil
}

// WriteHashRedis 向redis写hashmap数据
func (fw *WMFrameWorkV2) WriteHashRedis(key string, hashes map[string]string) error {
	if !fw.redisCtl.enable {
		return fmt.Errorf("redis is not ready")
	}
	args := make([]string, len(hashes)*2)
	var idx = 0
	for f, v := range hashes {
		args[idx] = f
		args[idx+1] = v
		idx += 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	err := fw.redisCtl.client.HSet(ctx, fw.AppendRootPathRedis(key), args).Err()
	if err != nil {
		fw.WriteError("REDIS", "Failed write redis hashmap data: "+key+"|"+err.Error())
		return err
	}
	return nil
}

// HDel 删redis
func (fw *WMFrameWorkV2) DelHashRedis(key string, fields ...string) error {
	if !fw.redisCtl.enable {
		return fmt.Errorf("redis is not ready")
	}

	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	err := fw.redisCtl.client.HDel(ctx, fw.AppendRootPathRedis(key), fields...).Err()
	if err != nil {
		fw.WriteError("REDIS", fmt.Sprintf("Failed erase redis data: %+v|%s", key, err.Error()))
		return err
	}
	fw.WriteInfo("REDIS", fmt.Sprintf("Erase redis data:%+v", key))
	return nil
}

// ReadAllRedisKeys 模糊读取所有匹配的key
func (fw *WMFrameWorkV2) ReadAllRedisKeys(key string) *redis.StringSliceCmd {
	if !fw.redisCtl.enable {
		return &redis.StringSliceCmd{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	return fw.redisCtl.client.Keys(ctx, fw.AppendRootPathRedis(key))
}

// ReadAllRedis 模糊读redis
func (fw *WMFrameWorkV2) ReadAllRedis(key string) ([]string, error) {
	if !fw.redisCtl.enable {
		return []string{}, fmt.Errorf("redis is not ready")
	}
	key = fw.AppendRootPathRedis(key)
	ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
	defer cancel()
	val := fw.redisCtl.client.Keys(ctx, key)
	if val.Err() != nil {
		fw.WriteError("REDIS", "Failed read redis data: "+key+"|"+val.Err().Error())
		return []string{}, val.Err()
	}
	var s = make([]string, 0)
	for _, v := range val.Val() {
		ctx, cancel := context.WithTimeout(context.Background(), redisCtxTimeo)
		defer cancel()
		vv := fw.redisCtl.client.Get(ctx, v)
		if vv.Err() == nil {
			s = append(s, vv.Val())
		}
	}
	return s, nil
}

// 返回redis客户端
func (fw *WMFrameWorkV2) RedisClient() *redis.Client {
	return fw.redisCtl.client
}

// RedisIsReady 返回redis可用状态
func (fw *WMFrameWorkV2) RedisIsReady() bool {
	return fw.redisCtl.enable
}

// ViewRedisConfig 查看redis配置,返回json字符串
func (fw *WMFrameWorkV2) ViewRedisConfig() string {
	return fw.redisCtl.forshow
}

// ExpireUserToken 更新token有效期
func (fw *WMFrameWorkV2) ExpireUserToken(token string) {
	// 更新redis的对应键值的有效期
	go fw.ExpireRedis("usermanager/legal/"+MD5Worker.Hash([]byte(token)), fw.tokenLife)
}
