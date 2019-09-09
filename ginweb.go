package wlstmicro

import (
	"github.com/gin-gonic/gin"
	"github.com/gogo/protobuf/proto"
	"github.com/tidwall/gjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/db"
	ginmiddleware "github.com/xyzj/gopsu/gin-middleware"
)

// NewGinEngine 返回一个新的gin路由
// logName：日志文件名
// logDays：日志保留天数
// logLevel：日志等级
// logGZ：是否压缩归档日志
// debug：是否使用调试模式
func NewGinEngine(logDir, logName string, logDays, logLevel int) *gin.Engine {
	return ginmiddleware.NewGinEngine(logDir, logName, logDays, logLevel)
}

// ListenAndServe 启用监听
// port：端口号
// timeout：读写超时
// h： http.hander, like gin.New()
func ListenAndServe(port int, h *gin.Engine) error {
	return ginmiddleware.ListenAndServe(port, h)
}

// ListenAndServeTLS 启用TLS监听
// port：端口号
// timeout：读写超时
// h： http.hander, like gin.New()
// certfile： cert file path
// keyfile： key file path
// clientca: 客户端根证书用于验证客户端合法性
func ListenAndServeTLS(port int, h *gin.Engine, certfile, keyfile string, clientca ...string) error {
	return ginmiddleware.ListenAndServeTLS(port, h, certfile, keyfile, clientca...)
}

// CheckRequired 检查必填参数
func CheckRequired(params ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, v := range params {
			if c.Param(v) == "" {
				c.Set("status", 0)
				c.Set("detail", "Missing parameters: "+v)
				c.AbortWithStatusJSON(200, c.Keys)
				return
			}
		}
		c.Next()
	}
}

// ReadCacheJSON 读取数据库缓存
func ReadCacheJSON(mydb *db.MySQL) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mydb != nil {
			cachetag := c.Param("cachetag")
			if cachetag != "" {
				if gopsu.IsExist(cachetag) {
					cachestart := gopsu.String2Int(c.Param("cachestart"), 10)
					cacherows := gopsu.String2Int(c.Param("cachesrows"), 10)
					ans := mydb.QueryCacheJSON(cachetag, cachestart, cacherows)
					if gjson.Parse(ans).Get("total").Int() > 0 {
						c.Params = append(c.Params, gin.Param{
							Key:   "_cacheData",
							Value: ans,
						})
					}
				}
			}
		}
		c.Next()
	}
}

// ReadCachePB2 读取数据库缓存
func ReadCachePB2(mydb *db.MySQL) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mydb != nil {
			cachetag := c.Param("cachetag")
			if cachetag != "" {
				if gopsu.IsExist(cachetag) {
					cachestart := gopsu.String2Int(c.Param("cachestart"), 10)
					cacherows := gopsu.String2Int(c.Param("cachesrows"), 10)
					ans := mydb.QueryCachePB2(cachetag, cachestart, cacherows)
					if ans.Total > 0 {
						b, _ := proto.Marshal(ans)
						c.Params = append(c.Params, gin.Param{
							Key:   "_cacheData",
							Value: string(b),
						})
					}
				}
			}
		}
		c.Next()
	}
}
