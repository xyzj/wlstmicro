package wlstmicro

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	gingzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/tidwall/gjson"
	"github.com/xyzj/gopsu"
	ginmiddleware "github.com/xyzj/gopsu/gin-middleware"
)

// NewHTTPEngine 创建gin引擎
func NewHTTPEngine(f ...gin.HandlerFunc) *gin.Engine {
	if !flag.Parsed() {
		flag.Parse()
	}
	r := gin.New()
	// 中间件
	//cors
	r.Use(cors.New(cors.Config{
		MaxAge:           time.Hour * 24,
		AllowAllOrigins:  true,
		AllowCredentials: true,
		AllowWildcard:    true,
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
	}))
	// 数据压缩
	r.Use(gingzip.Gzip(9))
	// 日志
	logName := ""
	if *logLevel > 0 {
		logName = fmt.Sprintf("X%d.http", *WebPort)
	}
	r.Use(ginmiddleware.LoggerWithRolling(gopsu.DefaultLogDir, logName, *logDays))
	// 错误恢复
	r.Use(ginmiddleware.Recovery())
	// 其他中间件
	if f != nil {
		r.Use(f...)
	}
	// 读取请求参数
	// r.Use(ginmiddleware.ReadParams())
	// 渲染模板
	// r.HTMLRender = multiRender()
	// 基础路由
	// 404,405
	r.HandleMethodNotAllowed = true
	r.NoMethod(ginmiddleware.Page405)
	r.NoRoute(ginmiddleware.Page404)
	r.GET("/", ginmiddleware.PageDefault)
	r.POST("/", ginmiddleware.PageDefault)
	r.GET("/health", ginmiddleware.PageDefault)
	r.GET("/clearlog", ginmiddleware.CheckRequired("name"), ginmiddleware.Clearlog)
	r.GET("/runtime", ginmiddleware.PageRuntime)
	r.Static("/static", filepath.Join(gopsu.GetExecDir(), "static"))
	return r
}

// NewHTTPService 启动HTTP服务
func NewHTTPService(r *gin.Engine) {
	var sss string
	for _, v := range r.Routes() {
		if v.Path == "/" || v.Method == "HEAD" || strings.HasSuffix(v.Path, "*filepath") || strings.HasPrefix(v.Path, "/proxy") || strings.HasPrefix(v.Path, "/plain") {
			continue
		}
		if strings.ContainsAny(v.Path, "*") && !strings.HasSuffix(v.Path, "filepath") {
			continue
		}
		sss += fmt.Sprintf(`<a>%s: %s</a><br><br>`, v.Method, v.Path)
	}
	if sss != "" {
		r.GET("/showroutes", func(c *gin.Context) {
			c.Header("Content-Type", "text/html")
			c.Status(http.StatusOK)
			render.WriteString(c.Writer, sss, nil)
		})
	}
	go func() {
		var err error
		if *Debug || *forceHTTP {
			err = ginmiddleware.ListenAndServe(*WebPort, r)
		} else {
			err = ginmiddleware.ListenAndServeTLS(*WebPort, r, HTTPTLS.Cert, HTTPTLS.Key, HTTPTLS.ClientCA)
		}
		if err != nil {
			WriteError("HTTP", "Failed start HTTP(S) server at :"+strconv.Itoa(*WebPort)+"|"+err.Error())
		}
	}()
}

// DoRequest 进行http request请求
// req: http.NewRequest()
// logdetail: [日志等级(0,10,20,30,40),日志追加信息]
// 返回statusCode, body, headers, error
func DoRequest(req *http.Request, logdetail ...string) (int, []byte, map[string]string, error) {
	level := 20
	if len(logdetail) > 0 {
		switch logdetail[0] {
		case "nil":
			level = 0
		case "debug":
			level = 10
		case "info":
			level = 20
		case "warn":
			level = 30
		case "error":
			level = 40
		}
	}
	WriteLog("HTTP", fmt.Sprintf("%s request to %s|%s", req.Method, req.URL.String(), strings.Join(logdetail, ",")), level)
	resp, err := HTTPClient.Do(req)
	if err != nil {
		WriteError("HTTP", "request error: "+err.Error())
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	_, err = b.ReadFrom(resp.Body)
	if err != nil {
		WriteError("HTTP", "read body error: "+err.Error())
		return 0, nil, nil, err
	}
	h := make(map[string]string)
	for k := range resp.Header {
		h[k] = resp.Header.Get(k)
	}
	WriteLog("HTTP", fmt.Sprintf("%s response %d from %s|%v", req.Method, resp.StatusCode, req.URL.String(), b.String()), level)
	return resp.StatusCode, b.Bytes(), h, nil
}

// PrepareToken 获取User-Token信息
// forceAbort: token非法时是否退出接口，true-退出，false-不退出
func PrepareToken(forceAbort ...bool) gin.HandlerFunc {
	shouldAbort := false
	if len(forceAbort) > 0 {
		shouldAbort = forceAbort[0]
	}
	return func(c *gin.Context) {
		uuid := c.GetHeader("User-Token")
		if len(uuid) != 36 {
			if shouldAbort {
				c.Set("status", 0)
				c.Set("detail", "User-Token illegal")
				c.AbortWithStatusJSON(200, c.Keys)
			}
			return
		}
		tokenPath := AppendRootPathRedis("usermanager/legal/" + MD5Worker.Hash([]byte(uuid)))
		x, err := ReadRedis(tokenPath)
		if err != nil {
			if shouldAbort {
				c.Set("status", 0)
				c.Set("detail", "User-Token illegal")
				c.AbortWithStatusJSON(200, c.Keys)
			}
			return
		}
		ans := gjson.Parse(x)
		if !ans.Exists() {
			if shouldAbort {
				c.Set("status", 0)
				c.Set("detail", "User-Token illegal")
				c.AbortWithStatusJSON(200, c.Keys)
			}
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
			Value: strings.Join(enableapi, ","),
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
