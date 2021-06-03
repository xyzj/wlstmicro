package wmv2

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/sjson"

	"github.com/gin-contrib/cors"
	gingzip "github.com/gin-contrib/gzip"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/tidwall/gjson"
	"github.com/xyzj/gopsu"
	game "github.com/xyzj/gopsu/games"
	ginmiddleware "github.com/xyzj/gopsu/gin-middleware"
	yaaggin "github.com/xyzj/yaag/gin"
	"github.com/xyzj/yaag/yaag"
)

const (
	TPLHEAD = `<html lang="zh-cn">
<head>
<meta content="text/html; charset=utf-8" http-equiv="content-type" />
{{template "css"}}
</head>
{{template "body" .}}
</html>`
	TPLCSS = `{{define "css"}}
<style type="text/css">
a {
  color: #4183C4;
  font-size: 16px; }
h1, h2, h3, h4, h5, h6 {
  margin: 20px 0 10px;
  padding: 0;
  font-weight: bold;
  -webkit-font-smoothing: antialiased;
  cursor: text;
  position: relative; }
h1 {
  font-size: 28px;
  color: black; }
h2 {
  font-size: 24px;
  border-bottom: 1px solid #cccccc;
  color: black; }
h3 {
  font-size: 18px; }
h4 {
  font-size: 16px; }
h5 {
  font-size: 14px; }
h6 {
  color: #777777;
  font-size: 14px; }
table {
  padding: 0; }
	table tr {
	  border-top: 1px solid #cccccc;
	  background-color: white;
	  margin: 0;
	  padding: 0; }
	  table tr:nth-child(2n) {
		background-color: #f8f8f8; }
	  table tr th {
		font-weight: bold;
		border: 1px solid #cccccc;
		text-align: center;
		margin: 0;
		padding: 6px 13px; }
	  table tr td {
		border: 1px solid #cccccc;
		text-align: left;
		margin: 0;
		padding: 6px 13px; }
	  table tr th :first-child, table tr td :first-child {
		margin-top: 0; }
	  table tr th :last-child, table tr td :last-child {
		margin-bottom: 0; }
</style>
{{end}}`
	TPLBODY = `{{define "body"}}
<body>
<h3>服务器系统时间：</h3><a>{{.timer}}</a>
<h3>服务启动时间：</h3><a>{{.startat}}</a>
<h3>{{.key}}：</h3><a>{{range $idx, $elem := .value}}
{{$elem}}<br>
{{end}}</a>
</body>
</html>
{{end}}`
)

//go:embed yaag
var apirec embed.FS

var (
	apidocPath = "docs/apidoc.html"
	yaagConfig *yaag.Config
	rever      = strings.NewReplacer("{\n", "", "}", "", `"`, "", ",", "")
)

func apidoc(c *gin.Context) {
	switch c.Param("switch") {
	case "on":
		yaagConfig.On = true
		c.String(200, "API record is set to on.")
	case "off":
		yaagConfig.On = false
		c.String(200, "API record is set to off.")
	case "reset":
		yaagConfig.ResetDoc()
		c.String(200, "API record reset done.")
	default:
		p := gopsu.JoinPathFromHere("docs", "apirecord-"+c.Param("switch")+".html")
		if gopsu.IsExist(p) {
			b, _ := ioutil.ReadFile(p)
			c.Header("Content-Type", "text/html")
			c.Status(http.StatusOK)
			c.Writer.Write(b)
		} else {
			c.String(200, "The API record file was not found, you may not have the API record function turned on.")
		}
	}
}

// NewHTTPEngine 创建gin引擎
func (fw *WMFrameWorkV2) NewHTTPEngine(f ...gin.HandlerFunc) *gin.Engine {
	if !*debug {
		gin.SetMode(gin.ReleaseMode)
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
	if *logLevel > 1 {
		logName = fw.loggerMark + ".http"
	}
	r.Use(ginmiddleware.LoggerWithRolling(gopsu.DefaultLogDir, logName, *logDays))
	// 错误恢复
	r.Use(ginmiddleware.Recovery())
	// 其他中间件
	if f != nil {
		r.Use(f...)
	}
	// 基础路由
	// 404,405
	r.HandleMethodNotAllowed = true
	r.NoMethod(ginmiddleware.Page405)
	r.NoRoute(ginmiddleware.Page404)
	r.GET("/whoami", func(c *gin.Context) {
		c.String(200, c.ClientIP())
	})
	r.GET("/devquotes", ginmiddleware.Page500)
	r.GET("/health", ginmiddleware.PageDefault)
	r.GET("/health/mod", fw.pageModCheck)
	r.POST("/health/mod", fw.pageModCheck)
	r.GET("/clearlog", ginmiddleware.CheckRequired("name"), ginmiddleware.Clearlog)
	r.GET("/status", fw.pageStatus)
	r.POST("/status", fw.pageStatus)
	r.GET("/viewconfig", func(c *gin.Context) {
		configInfo := make(map[string]interface{})
		configInfo["startat"] = fw.startAt
		configInfo["timer"] = time.Now().Format("2006-01-02 15:04:05 Mon")
		configInfo["key"] = "服务配置信息"
		b, _ := ioutil.ReadFile(fw.wmConf.FullPath())
		configInfo["value"] = strings.Split(string(b), "\n")
		c.Header("Content-Type", "text/html")
		t, _ := template.New("viewconfig").Parse(TPLHEAD + TPLCSS + TPLBODY)
		h := render.HTML{
			Name:     "viewconfig",
			Data:     configInfo,
			Template: t,
		}
		h.WriteContentType(c.Writer)
		h.Render(c.Writer)
	})
	r.Static("/static", gopsu.JoinPathFromHere("static"))
	// apirecord
	r.StaticFS("/apirec", http.FS(apirec))
	r.GET("/apirecord/:switch", apidoc)
	// 生成api访问文档
	apidocPath = gopsu.JoinPathFromHere("docs", "apirecord-"+fw.serverName+".html")
	os.MkdirAll(gopsu.JoinPathFromHere("docs"), 0755)
	yaagConfig = &yaag.Config{
		On:       false,
		DocTitle: "Gin Web Framework API Record",
		DocPath:  apidocPath,
		BaseUrls: map[string]string{
			"Server Name": fw.serverName,
			"Core Author": "X.Yuan",
			"Last Update": time.Now().Format(gopsu.LongTimeFormat),
		},
	}
	yaag.Init(yaagConfig)
	r.Use(yaaggin.Document())
	// have fun
	// r.GET("/game", game.GameGroup)
	r.GET("/game/:game", game.GameGroup)
	return r
}

// NewHTTPService 启动HTTP服务
func (fw *WMFrameWorkV2) newHTTPService(r *gin.Engine) {
	var sss string
	var findRoot bool
	for _, v := range r.Routes() {
		if v.Path == "/" {
			findRoot = true
			continue
		}
		if v.Method == "HEAD" || strings.HasSuffix(v.Path, "*filepath") || strings.HasPrefix(v.Path, "/proxy") || strings.HasPrefix(v.Path, "/plain") {
			continue
		}
		if strings.ContainsAny(v.Path, "*") && !strings.HasSuffix(v.Path, "filepath") {
			continue
		}
		sss += fmt.Sprintf(`<a>%s: %s</a><br><br>`, v.Method, v.Path)
	}
	if !findRoot {
		r.GET("/", ginmiddleware.PageDefault)
	}
	if sss != "" {
		r.GET("/showroutes", func(c *gin.Context) {
			c.Header("Content-Type", "text/html")
			c.Status(http.StatusOK)
			render.WriteString(c.Writer, sss, nil)
		})
	}

	var err error
	if *debug || *forceHTTP {
		fw.httpProtocol = "http://"
		err = ginmiddleware.ListenAndServe(*webPort, r)
	} else {
		fw.httpProtocol = "https://"
		err = ginmiddleware.ListenAndServeTLS(*webPort, r, fw.httpCert, fw.httpKey, "")
	}
	if err != nil {
		fw.WriteError("HTTP", "Failed start HTTP(S) server at :"+strconv.Itoa(*webPort)+"|"+err.Error())
	}
}

// DoRequest 进行http request请求
// req: http.NewRequest()
// logdetail: [日志等级(0,10,20,30,40),日志追加信息]
// 返回statusCode, body, headers, error
func (fw *WMFrameWorkV2) DoRequest(req *http.Request, logdetail ...string) (int, []byte, map[string]string, error) {
	level := 20
	if !*debug {
		if len(logdetail) == 0 || logdetail[0] == "nil" {
			level = 0
		}
	}
	// fw.WriteLog("HTTP", fmt.Sprintf("%s request to %s|%s", req.Method, req.URL.String(), strings.Join(logdetail, ",")), 10)
	resp, err := fw.httpClientPool.Do(req)
	if err != nil {
		fw.WriteError("HTTP FWD", "request error: "+err.Error())
		return 502, nil, nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fw.WriteError("HTTP FWD", "read body error: "+err.Error())
		return 502, nil, nil, err
	}
	h := make(map[string]string)
	h["resp_from"] = req.Host
	for k := range resp.Header {
		h[k] = resp.Header.Get(k)
	}
	sc := resp.StatusCode
	fw.WriteLog("HTTP FWD", fmt.Sprintf("%s response %d from %s|%v", req.Method, sc, req.URL.String(), string(b)), level)
	return sc, b, h, nil
}

func (fw *WMFrameWorkV2) pageModCheck(c *gin.Context) {
	var tbody = `{{define "body"}}
<body>
<h3>服务器时间：</h3><a>{{.timer}}</a>
<h3>服务模块状态：</h3>
<table>
<thead>
<tr>
<th>启用的模块</th>
<th>模块状态</th>
</tr>
</thead>
<tbody>
	{{range $idx, $elem := .clients}}
	<tr>
		{{range $key,$value:=$elem}}
			<td>{{$value}}</td>
		{{end}}
	</tr>
	{{end}}
</tbody>
</table>
</body>
{{end}}`
	var serviceCheck = make([][]string, 0)
	// 版本
	serviceCheck = append(serviceCheck, []string{"ver", gjson.Parse(fw.verJSON).Get("version").String()})
	// 检查etc
	serviceCheck = append(serviceCheck, []string{"etcd", func() string {
		if fw.cnf.UseETCD == nil || !fw.cnf.UseETCD.Activation {
			return "---"
		}
		if _, err := fw.Picker(fw.serverName); err != nil {
			return "bad"
		}
		return "ok"
	}()})
	// 检查redis
	serviceCheck = append(serviceCheck, []string{"redis", func() string {
		if fw.cnf.UseRedis == nil || !fw.cnf.UseRedis.Activation {
			return "---"
		}
		if err := fw.WriteRedis(gopsu.GetUUID1(), "value interface{}", time.Second); err != nil {
			return "bad"
		}
		return "ok"
	}()})
	// 检查mq生产者
	serviceCheck = append(serviceCheck, []string{"mq producer", func() string {
		if fw.cnf.UseMQProducer == nil || !fw.cnf.UseMQProducer.Activation {
			return "---"
		}
		if !fw.ProducerIsReady() {
			return "bad"
		}
		return "ok"
	}()})
	// 检查mq消费者
	serviceCheck = append(serviceCheck, []string{"mq consumer", func() string {
		if fw.cnf.UseMQConsumer == nil || !fw.cnf.UseMQConsumer.Activation {
			return "---"
		}
		if !fw.ConsumerIsReady() {
			return "bad"
		}
		return "ok"
	}()})
	// 检查sql
	serviceCheck = append(serviceCheck, []string{"sql", func() string {
		if fw.cnf.UseSQL == nil || !fw.cnf.UseSQL.Activation {
			return "---"
		}
		if !fw.MysqlIsReady() {
			return "bad"
		}
		return "ok"
	}()})
	// 检查tcp
	serviceCheck = append(serviceCheck, []string{"tcp", func() string {
		if fw.cnf.UseTCP == nil || !fw.cnf.UseTCP.Activation {
			return "---"
		}
		if !fw.tcpCtl.enable {
			return "bad"
		}
		return "ok"
	}()})
	if c.Request.Method == "GET" {
		var d = gin.H{
			"timer":   gopsu.Stamp2Time(time.Now().Unix()),
			"clients": serviceCheck,
		}
		t, _ := template.New("modcheck").Parse(TPLHEAD + TPLCSS + tbody)
		h := render.HTML{
			Name:     "modcheck",
			Data:     d,
			Template: t,
		}
		h.WriteContentType(c.Writer)
		h.Render(c.Writer)
		return
	}
	var js string
	for _, v := range serviceCheck {
		js, _ = sjson.Set(js, v[0], v[1])
	}
	c.PureJSON(200, gjson.Parse(js).Value())
}

func (fw *WMFrameWorkV2) pageStatus(c *gin.Context) {
	var statusInfo = make(map[string]interface{})
	statusInfo["startat"] = fw.startAt
	statusInfo["timer"] = time.Now().Format("2006-01-02 15:04:05 Mon")
	statusInfo["key"] = "服务运行信息"
	fmtver, _ := json.MarshalIndent(gjson.Parse(fw.verJSON).Value(), "", "")
	statusInfo["value"] = strings.Split(rever.Replace(string(fmtver)), "\n")
	switch c.Request.Method {
	case "GET":
		c.Header("Content-Type", "text/html")
		t, _ := template.New("runtime").Parse(TPLHEAD + TPLCSS + TPLBODY)
		h := render.HTML{
			Name:     "runtime",
			Data:     statusInfo,
			Template: t,
		}
		h.WriteContentType(c.Writer)
		h.Render(c.Writer)
	case "POST":
		c.Set("server_time", statusInfo["timer"].(string))
		c.Set("start_at", statusInfo["startat"].(string))
		c.Set("ver", gjson.Parse(fw.verJSON).Value())
		c.Set("conf", gjson.Parse(fw.wmConf.GetAll()).Value())
		c.PureJSON(200, c.Keys)
	}
}

// PrepareToken 获取User-Token信息
// forceAbort: token非法时是否退出接口，true-退出，false-不退出
func (fw *WMFrameWorkV2) PrepareToken(forceAbort ...bool) gin.HandlerFunc {
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
				c.AbortWithStatusJSON(http.StatusUnauthorized, c.Keys)
			}
			return
		}
		tokenPath := fw.AppendRootPathRedis("usermanager/legal/" + MD5Worker.Hash([]byte(uuid)))
		x, err := fw.ReadRedis(tokenPath)
		if err != nil {
			if shouldAbort {
				c.Set("status", 0)
				c.Set("detail", "User-Token not found")
				c.AbortWithStatusJSON(http.StatusUnauthorized, c.Keys)
			}
			return
		}
		ans := gjson.Parse(x)
		if !ans.Exists() { // token内容异常
			if shouldAbort {
				c.Set("status", 0)
				c.Set("detail", "User-Token can not understand")
				c.AbortWithStatusJSON(http.StatusUnauthorized, c.Keys)
			}
			fw.EraseRedis(tokenPath)
			return
		}
		if ans.Get("expire").Int() > 0 && ans.Get("expire").Int() < time.Now().Unix() { // 用户过期
			if shouldAbort {
				c.Set("status", 0)
				c.Set("detail", "Account has expired")
				c.AbortWithStatusJSON(http.StatusUnauthorized, c.Keys)
			}
			fw.EraseRedis(tokenPath)
			return
		}
		c.Params = append(c.Params, gin.Param{
			Key:   "_userTokenPath",
			Value: tokenPath,
		})
		c.Params = append(c.Params, gin.Param{
			Key:   "_userDepID",
			Value: ans.Get("dep_id").String(),
		})
		c.Params = append(c.Params, gin.Param{
			Key:   "_userTokenName",
			Value: ans.Get("user_name").String(),
		})
		asadmin := ans.Get("asadmin").String()
		if asadmin == "0" {
			asadmin = ans.Get("userinfo.User_admin").String()
		}
		c.Params = append(c.Params, gin.Param{
			Key:   "_userAsAdmin",
			Value: asadmin,
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
		// // 更新redis的对应键值的有效期
		// if ans.Get("source").String() != "local" {
		// 	fw.ExpireUserToken(uuid)
		// }
	}
}

// RenewToken 更新uuid时效
func (fw *WMFrameWorkV2) RenewToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		uuid := c.GetHeader("User-Token")
		if len(uuid) != 36 {
			return
		}
		x, err := fw.ReadRedis("usermanager/legal/" + MD5Worker.Hash([]byte(uuid)))
		if err != nil {
			return
		}
		// 更新redis的对应键值的有效期
		if gjson.Parse(x).Get("source").String() != "local" {
			fw.ExpireUserToken(uuid)
		}
	}
}

// GoUUID 获取特定uuid
func (fw *WMFrameWorkV2) GoUUID(uuid, username string) (string, bool) {
	if len(uuid) == 36 {
		return uuid, true
	}
	addr, err := fw.PickerDetail("usermanager")
	if err != nil {
		fw.WriteError("ETCD", "can not found server usermanager")
		return "", false
	}
	var req *http.Request
	req, _ = http.NewRequest("GET", addr+"/usermanager/v1/user/fixed/login?user_name="+username, strings.NewReader(""))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Legal-High", gopsu.CalculateSecurityCode("m", time.Now().Month().String(), 0)[0])
	resp, err := fw.httpClientPool.Do(req)
	if err != nil {
		fw.WriteError("CORE", "get uuid error:"+err.Error())
		return "", false
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	_, err = b.ReadFrom(resp.Body)
	if err != nil {
		fw.WriteError("CORE", "read uuid error:"+err.Error())
		return "", false
	}
	return b.String(), true
}

// DealWithSQLError 统一处理sql执行错误问题
func (fw *WMFrameWorkV2) DealWithSQLError(c *gin.Context, err error) bool {
	if err != nil {
		fw.WriteError("SQL", c.Request.RequestURI+"|"+err.Error())
		c.Set("status", 0)
		c.Set("detail", "sql error")
		c.Set("xfile", 3)
		c.PureJSON(500, c.Keys)
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

// SetTokenLife 设置User-Token的有效期，默认30分钟
func (fw *WMFrameWorkV2) SetTokenLife(t time.Duration) {
	fw.tokenLife = t
}
