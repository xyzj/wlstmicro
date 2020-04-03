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

// CheckUUID 通过uuid获取用户信息
func CheckUUID(c *gin.Context) {
	uuid := c.GetHeader("User-Token")
	x, _ := ReadRedis("usermanager/legal/" + gopsu.GetMD5(uuid))
	if len(x) == 0 {
		c.Set("status", 0)
		c.Set("detail", "User-Token illegal")
		c.AbortWithStatusJSON(200, c.Keys)
		return
	}
	ans := gjson.Parse(x)
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
	go func() {
		defer func() {
			if err := recover(); err != nil {
				WriteError("REDIS", err.(error).Error())
			}
		}()
		if ans.Get("source").String() != "local" {
			ExpireRedis("usermanager/legal/"+gopsu.GetMD5(uuid), time.Minute*30)
		}
	}()
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
