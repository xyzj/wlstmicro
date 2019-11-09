package wlstmicro

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/xyzj/gopsu"
)

// CheckUUID 通过uuid获取用户信息
func CheckUUID() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid := c.GetHeader("User-Token")
		x, err := ReadRedis("usermanager/legal/" + gopsu.GetMD5(uuid))
		if err != nil { // redis读取失败，从usermanager里查询
			s, err := PickerDetail("usermanager")
			if err != nil {
				c.Set("status", 0)
				c.Set("detail", "User-Token illegal |"+err.Error())
				c.AbortWithStatusJSON(200, c.Keys)
				return
			}
			s += "/usermanager/v1/user/verify?uuid=" + uuid
			resp, err := http.Get(s)
			if err != nil {
				c.Set("status", 0)
				c.Set("detail", "User-Token illegal |"+err.Error())
				c.AbortWithStatusJSON(200, c.Keys)
				return
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				c.Set("status", 0)
				c.Set("detail", "User-Token illegal |"+err.Error())
				c.AbortWithStatusJSON(200, c.Keys)
				return
			}
			x = string(b)
		}
		if len(x) == 0 {
			c.Set("status", 0)
			c.Set("detail", "User-Token illegal")
			c.AbortWithStatusJSON(200, c.Keys)
			return
		}
		if time.Now().Unix() > gjson.Parse(x).Get("expire").Int() {
			c.Set("status", 0)
			c.Set("detail", "user expired")
			c.AbortWithStatusJSON(200, c.Keys)
			return
		}
		c.Params = append(c.Params, gin.Param{
			Key:   "_userTokenName",
			Value: gjson.Parse(x).Get("user_name").String(),
		})
	}
}
