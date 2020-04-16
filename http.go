package wlstmicro

import (
	"bytes"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/xyzj/gopsu"
)

// PrepareToken 获取User-Token信息
func PrepareToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid := c.GetHeader("User-Token")
		if len(uuid) != 36 {
			return
		}
		tokenPath := AppendRootPathRedis("usermanager/legal/" + MD5Worker.Hash([]byte(uuid)))
		x, err := ReadRedis(tokenPath)
		if err != nil {
			return
		}
		ans := gjson.Parse(x)
		if !ans.Exists() {
			return
		}
		c.Params = append(c.Params, gin.Param{
			Key:   "_userTokenPath",
			Value: tokenPath,
		})
		c.Params = append(c.Params, gin.Param{
			Key:   "_userTokenName",
			Value: ans.Get("user_name").String(),
		})
		c.Params = append(c.Params, gin.Param{
			Key:   "_userAsAdmin",
			Value: ans.Get("asadmin").String(),
		})
		c.Params = append(c.Params, gin.Param{
			Key:   "_userRoleID",
			Value: ans.Get("role_id").String(),
		})
		authbinding := make([]string, 0)
		for _, v := range ans.Get("auth_binding").Array() {
			authbinding = append(authbinding, v.String())
		}
		c.Params = append(c.Params, gin.Param{
			Key:   "_authBinding",
			Value: strings.Join(authbinding, ","),
		})
		enableapi := make([]string, 0)
		for _, v := range ans.Get("enable_api").Array() {
			enableapi = append(enableapi, v.String())
		}
		c.Params = append(c.Params, gin.Param{
			Key:   "_enableAPI",
			Value: strings.Join(authbinding, ","),
		})
		// 更新redis的对应键值的有效期
		if ans.Get("source").String() != "local" {
			ExpireUserToken(uuid)
		}
	}
}

// RenewToken 更新uuid时效
func RenewToken(c *gin.Context) {
	uuid := c.GetHeader("User-Token")
	if len(uuid) != 36 {
		return
	}
	x, err := ReadRedis("usermanager/legal/" + MD5Worker.Hash([]byte(uuid)))
	if err != nil {
		return
	}
	// 更新redis的对应键值的有效期
	if gjson.Parse(x).Get("source").String() != "local" {
		ExpireUserToken(uuid)
	}
}

// CheckToken 通过uuid获取用户信息
func CheckToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Param("_userTokenPath") == "" {
			c.Set("status", 0)
			c.Set("detail", "User-Token illegal")
			c.Set("xfile", 11)
			c.AbortWithStatusJSON(401, c.Keys)
		}
	}
}

// GoUUID 获取特定uuid
func GoUUID(uuid, username string) (string, bool) {
	if len(uuid) == 36 {
		return uuid, true
	}
	addr, err := PickerDetail("usermanager")
	if err != nil {
		WriteError("ETCD", "can not found server usermanager")
		return "", false
	}
	var req *http.Request
	req, _ = http.NewRequest("GET", addr+"/usermanager/v1/user/fixed/login?user_name="+username, strings.NewReader(""))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Legal-High", gopsu.CalculateSecurityCode("m", time.Now().Month().String(), 0)[0])
	resp, err := HTTPClient.Do(req)
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

// DealWithSQLError 统一处理sql执行错误问题
func DealWithSQLError(c *gin.Context, err error) bool {
	if err != nil {
		WriteError("SQL", c.Request.RequestURI+"|"+err.Error())
		c.Set("status", 0)
		c.Set("detail", "sql error")
		c.Set("xfile", 3)
		c.PureJSON(200, c.Keys)
		return true
	}
	return false
}

// JSON2Key json字符串赋值到gin.key
func JSON2Key(c *gin.Context, s string) {
	gjson.Parse(s).ForEach(func(key, value gjson.Result) bool {
		c.Set(key.String(), value.Value())
		return true
	})
}
