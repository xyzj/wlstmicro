package wlstmicro

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"strings"
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

// GoUUID 获取特定uuid
func GoUUID(uuid, username string) (string, bool) {
	if len(uuid) == 36 {
		return uuid, true
	}
	addr, err := PickerDetail("usermanager")
	if err != nil {
		WriteError("CORE", "can not found server usermanager")
		return "", false
	}
	var client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: time.Duration(time.Second * 30),
	}
	var req *http.Request
	req, _ = http.NewRequest("GET", addr+"/usermanager/v1/user/fixed/login?user_name="+username, strings.NewReader(""))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Legal-High", gopsu.CalculateSecurityCode("m", time.Now().Month().String(), 0)[0])
	resp, err := client.Do(req)
	if err != nil {
		WriteError("CORE", "get uuid error:"+err.Error())
		return "", false
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	_, err = b.ReadFrom(resp.Body)
	if err != nil {
		WriteError("CORE", "read uuid error:"+err.Error())
		return "", false
	}
	return b.String(), true
}
