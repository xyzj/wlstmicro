package wmv2

import (
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/pyroscope-io/pyroscope/pkg/agent/profiler"
	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/db"
	msgctl "github.com/xyzj/proto/msgjk"
)

//go:embed ca/ca.pem
var ca []byte

//go:embed ca/localhost.pem
var caCert []byte

//go:embed ca/localhost-key.pem
var caKey []byte

//go:embed ca/localhost.pfx
var caPfx []byte

// NewFrameWorkV2 初始化一个新的framework
func NewFrameWorkV2(versionInfo string) *WMFrameWorkV2 {
	if !flag.Parsed() {
		flag.Parse()
	}
	if *help {
		flag.PrintDefaults()
		os.Exit(1)
	}
	fmtver, _ := json.MarshalIndent(gjson.Parse(versionInfo).Value(), "", "  ")
	if *ver {
		println(string(fmtver))
		os.Exit(1)
	}
	// 初始化
	fw := &WMFrameWorkV2{
		rootPath:      "wlst-micro",
		tokenLife:     time.Minute * 30,
		wmConf:        &gopsu.ConfData{},
		wmLog:         &gopsu.StdLogger{},
		serverName:    "X",
		startAt:       time.Now().Format("2006-01-02 15:04:05 Mon"),
		verJSON:       versionInfo,
		etcdCtl:       &etcdConfigure{},
		redisCtl:      &redisConfigure{},
		dbCtl:         &dbConfigure{},
		rmqCtl:        &rabbitConfigure{},
		tcpCtl:        &tcpConfigure{},
		chanTCPWorker: make(chan interface{}, 5000),
		JSON:          jsoniter.Config{}.Froze(),
		httpClientPool: &http.Client{
			Timeout: time.Duration(time.Second * 60),
			Transport: &http.Transport{
				IdleConnTimeout:     time.Second * 10,
				MaxConnsPerHost:     100,
				MaxIdleConns:        1,
				MaxIdleConnsPerHost: 1,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}
	// 处置版本，检查机器码
	fw.checkMachine()
	// 写版本信息
	p, _ := os.Executable()
	f, _ := os.OpenFile(p+".ver", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	defer f.Close()
	f.Write(fmtver)
	// 处置目录
	if *portable {
		gopsu.DefaultConfDir, gopsu.DefaultLogDir, gopsu.DefaultCacheDir = gopsu.MakeRuntimeDirs(".")
	} else {
		gopsu.DefaultConfDir, gopsu.DefaultLogDir, gopsu.DefaultCacheDir = gopsu.MakeRuntimeDirs("..")
	}
	// 日志
	if *debug {
		*logLevel = 10
	}
	if *logLevel <= 1 {
		*logDays = *logLevel
	}
	// 设置基础路径
	fw.baseCAPath = filepath.Join(gopsu.DefaultConfDir, "ca")
	if *capath != "" {
		fw.baseCAPath = *capath
	}
	if !gopsu.IsExist(fw.baseCAPath) {
		os.MkdirAll(fw.baseCAPath, 0755)
	}
	fw.tlsCert = filepath.Join(fw.baseCAPath, "localhost.pem")
	fw.tlsKey = filepath.Join(fw.baseCAPath, "localhost-key.pem")
	fw.tlsRoot = filepath.Join(fw.baseCAPath, "ca.pem")
	fw.httpCert = filepath.Join(fw.baseCAPath, "localhost.pem")
	fw.httpKey = filepath.Join(fw.baseCAPath, "localhost-key.pem")
	if !gopsu.IsExist(fw.tlsRoot) {
		ioutil.WriteFile(fw.tlsRoot, ca, 0644)
	}
	if !gopsu.IsExist(fw.tlsCert) {
		ioutil.WriteFile(fw.tlsCert, caCert, 0644)
	}
	if !gopsu.IsExist(fw.tlsKey) {
		ioutil.WriteFile(fw.tlsKey, caKey, 0644)
	}
	if !gopsu.IsExist(filepath.Join(fw.baseCAPath, "localhost.pfx")) {
		ioutil.WriteFile(filepath.Join(fw.baseCAPath, "localhost.pfx"), caPfx, 0644)
	}
	return fw
}

// Start 运行框架
// 启动模组，不阻塞
func (fw *WMFrameWorkV2) Start(opv2 *OptionFrameWorkV2) {
	// 设置日志
	fw.cnf = opv2
	if fw.loggerMark == "" {
		if opv2.UseETCD != nil {
			if opv2.UseETCD.SvrName != "" {
				fw.serverName = opv2.UseETCD.SvrName
			}
		}
		if fw.tcpCtl.bindPort > 0 {
			fw.loggerMark = fmt.Sprintf("%s-%05d", fw.serverName, fw.tcpCtl.bindPort)
		} else {
			fw.loggerMark = fmt.Sprintf("%s-%05d", fw.serverName, *webPort)
		}
	}
	fw.wmLog = gopsu.NewLogger(gopsu.DefaultLogDir, fw.loggerMark+".core", *logLevel, *logDays)
	if opv2.ConfigFile == "" {
		opv2.ConfigFile = *conf
	}
	// 载入配置
	if opv2.ConfigFile != "" {
		var cfpath string
		if strings.ContainsAny(opv2.ConfigFile, "\\/") {
			cfpath = opv2.ConfigFile
		} else {
			cfpath = filepath.Join(gopsu.DefaultConfDir, opv2.ConfigFile)
		}
		if !gopsu.IsExist(cfpath) {
			println("no config file found, try to create new one")
		}
		fw.loadConfigure(cfpath)
	}
	// 前置处理方法，用于预初始化某些内容
	if opv2.FrontFunc != nil {
		opv2.FrontFunc()
	}
	// etcd
	if opv2.UseETCD != nil {
		if opv2.UseETCD.SvrName != "" {
			fw.serverName = opv2.UseETCD.SvrName
		}
		if opv2.UseETCD.Activation {
			go fw.newETCDClient()
		}
	}
	// redis
	if opv2.UseRedis != nil {
		if opv2.UseRedis.Activation {
			fw.newRedisClient()
		}
	}
	// sql
	if opv2.UseSQL != nil {
		if opv2.UseSQL.Activation {
			if fw.newDBClient() {
				if opv2.UseSQL.DoMERGE {
					go fw.MaintainMrgTables()
				}
				// 检查是否存在更新的脚本
				exe, _ := os.Executable()
				upsql := exe + ".sql"
				if gopsu.IsExist(upsql) {
					b, err := ioutil.ReadFile(upsql)
					if err != nil {
						fw.WriteError("DBUP", err.Error())
					} else {
						var err error
						fw.WriteInfo("DBUP", "Try to update database by "+upsql)
						for _, v := range strings.Split(string(b), "\n") {
							s := gopsu.TrimString(v)
							if s == "" {
								continue
							}
							if _, _, err = fw.dbCtl.client.Exec(s); err != nil {
								fw.WriteError("DBUP", s+" | "+err.Error())
							}
						}
						if !*debug {
							os.Remove(upsql)
						}
					}
				}
			}
		}
	}
	// 生产者
	if opv2.UseMQProducer != nil {
		if opv2.UseMQProducer.Activation {
			fw.newMQProducer()
		}
	}
	// 消费者
	if opv2.UseMQConsumer != nil {
		if opv2.UseMQConsumer.Activation {
			if fw.newMQConsumer() {
				if opv2.UseMQConsumer.BindKeysFunc != nil {
					if ss, ok := opv2.UseMQConsumer.BindKeysFunc(); ok {
						opv2.UseMQConsumer.BindKeys = ss
					}
				}
				fw.BindRabbitMQ(opv2.UseMQConsumer.BindKeys...)
				go fw.recvRabbitMQ(opv2.UseMQConsumer.RecvFunc)
			}
		}
	}
	// tcp
	if opv2.UseTCP != nil {
		if opv2.UseTCP.Activation {
			if opv2.UseTCP.MQFlag == "" {
				opv2.UseTCP.MQFlag = "0"
			}
			fw.tcpCtl.mqFlag = opv2.UseTCP.MQFlag
			fw.tcpCtl.bindPort = opv2.UseTCP.BindPort
			go fw.newTCPService(opv2.UseTCP.Client)
		}
	}
	// gin http
	if opv2.UseHTTP != nil {
		if opv2.UseHTTP.Activation {
			if opv2.UseHTTP.EngineFunc == nil {
				opv2.UseHTTP.EngineFunc = func() *gin.Engine {
					return fw.NewHTTPEngine()
				}
			}
			go fw.newHTTPService(opv2.UseHTTP.EngineFunc())
		}
	}
	// gpstimer
	if fw.gpsTimer > 0 {
		go fw.newGPSConsumer()
	}
	// 执行额外方法
	if opv2.ExpandFuncs != nil {
		for _, v := range opv2.ExpandFuncs {
			v()
		}
	}
	// 启用性能调试，仅可用于开发过程中
	if *pyroscope {
		profiler.Start(profiler.Config{
			ApplicationName: fw.serverName + "_" + gopsu.RealIP(false) + "_" + gopsu.GetUUID1(),
			ServerAddress:   "http://office.shwlst.com:10097",
		})
	}
	fw.WriteSystem("", "Service start:"+fw.verJSON)
}

// Run 运行框架
// 启动模组，阻塞
func (fw *WMFrameWorkV2) Run(opv2 *OptionFrameWorkV2) {
	fw.Start(opv2)
	for {
		time.Sleep(time.Hour)
	}
}

// LoadConfigure 初始化配置
func (fw *WMFrameWorkV2) loadConfigure(f string) {
	var err error
	fw.wmConf, err = gopsu.LoadConfig(f)
	if err != nil {
		println("can not write config file")
	}
	fw.rootPath = fw.wmConf.GetItemDefault("root_path", "wlst-micro", "etcd/mq/redis注册根路径")
	fw.rootPathRedis = "/" + fw.rootPath + "/"
	fw.rootPathMQ = fw.rootPath + "."
	domainName := fw.wmConf.GetItemDefault("domain_name", "", "set the domain name, cert and key file name should be xxx.crt & xxx.key")
	fw.gpsTimer = gopsu.String2Int64(fw.wmConf.GetItemDefault("gpstimer", "0", "是否使用广播的gps时间进行对时操作,0-不启用，1-启用（30～900s内进行矫正），2-忽略误差范围强制矫正"), 10)

	fw.wmConf.Save()
	if domainName != "" {
		fw.httpCert = filepath.Join(fw.baseCAPath, domainName+".crt")
		fw.httpKey = filepath.Join(fw.baseCAPath, domainName+".key")
	}
	// 以下参数不自动生成，影响dorequest性能
	var trTimeo = time.Second * 60
	var trMaxconnPerHost int
	s, err := fw.wmConf.GetItem("tr_timeo")
	if err == nil {
		if gopsu.String2Int(s, 10) > 2 {
			trTimeo = time.Second * time.Duration(gopsu.String2Int(s, 10))
		}
	}
	s, err = fw.wmConf.GetItem("tr_maxconn_perhost")
	if err == nil {
		trMaxconnPerHost = gopsu.String2Int(s, 10)
	}
	if !(trMaxconnPerHost < 1 || trMaxconnPerHost > 2000) {
		trMaxconnPerHost = 100
	}
	fw.httpClientPool = &http.Client{
		Timeout: time.Duration(trTimeo),
		Transport: &http.Transport{
			IdleConnTimeout:     time.Second * 10,
			MaxConnsPerHost:     trMaxconnPerHost,
			MaxIdleConns:        1,
			MaxIdleConnsPerHost: 1,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

// GetLogger 返回日志模块
func (fw *WMFrameWorkV2) GetLogger() gopsu.Logger {
	return fw.wmLog
}

// ConfClient 配置文件实例
func (fw *WMFrameWorkV2) ConfClient() *gopsu.ConfData {
	return fw.wmConf
}

// ReadConfigItem 读取配置参数
func (fw *WMFrameWorkV2) ReadConfigItem(key, value, remark string) string {
	if fw.wmConf == nil {
		return ""
	}
	if value == "" {
		v, _ := fw.wmConf.GetItem(key)
		return v
	}
	return fw.wmConf.GetItemDefault(key, value, remark)
}

// ReadConfigKeys 获取配置所有key
func (fw *WMFrameWorkV2) ReadConfigKeys() []string {
	return fw.wmConf.GetKeys()
}

// ReadConfigAll 获取配置所有item
func (fw *WMFrameWorkV2) ReadConfigAll() string {
	return fw.wmConf.GetAll()
}

// ReloadConfig 重新读取
func (fw *WMFrameWorkV2) ReloadConfig() error {
	return fw.wmConf.Reload()
}

// WriteConfigItem 更新key
func (fw *WMFrameWorkV2) WriteConfigItem(key, value string) {
	fw.wmConf.UpdateItem(key, value)
}

// WriteConfig 读取配置参数
func (fw *WMFrameWorkV2) WriteConfig() {
	fw.wmConf.Save()
}

// WriteTCP 发送数据到tcp池
func (fw *WMFrameWorkV2) WriteTCP(v interface{}) {
	if v == nil {
		return
	}
	fw.chanTCPWorker <- v
}

// ReadTCPOnline 获取tcp在线信息
func (fw *WMFrameWorkV2) ReadTCPOnline() string {
	msg := &msgctl.MsgWithCtrl{}
	var ss, s string
	ss, _ = sjson.Set(ss, "timer", gopsu.Stamp2Time(time.Now().Unix()))
	if err := gopsu.JSON2PB(fw.tcpCtl.onlineSocks, msg); err == nil {
		for _, v := range msg.Syscmds.OnlineInfo {
			if v.PhyId > 0 {
				s, _ = sjson.Set("", "id", v.PhyId)
				s, _ = sjson.Set(s, "ip", v.Ip)
				s, _ = sjson.Set(s, "net", v.NetType)
				s, _ = sjson.Set(s, "ss", v.Signal)
				s, _ = sjson.Set(s, "imei", v.Imei)
				ss, _ = sjson.Set(ss, "clients.-1", s)
			}
		}
	}
	return ss
}

// Tag 版本标签
func (fw *WMFrameWorkV2) Tag() string {
	if fw.tag == "" {
		fw.tag = gjson.Parse(fw.verJSON).Get("version").String()
	}
	return fw.tag
}

// VersionInfo 获取版本信息
func (fw *WMFrameWorkV2) VersionInfo() string {
	return fw.verJSON
}

// WebPort http 端口
func (fw *WMFrameWorkV2) WebPort() int {
	return *webPort
}

// ServerName 服务名称
func (fw *WMFrameWorkV2) ServerName() string {
	return fw.serverName
}

// SetServerName 设置服务名称
func (fw *WMFrameWorkV2) SetServerName(s string) {
	fw.serverName = s
}

// SetLoggerMark 设置日志文件标识
func (fw *WMFrameWorkV2) SetLoggerMark(s string) {
	fw.loggerMark = s
}

// SetHTTPTimeout 设置http超时
func (fw *WMFrameWorkV2) SetHTTPTimeout(second int) {
	fw.httpClientPool.Timeout = time.Second * time.Duration(second)
}

// Debug 返回是否debug模式
func (fw *WMFrameWorkV2) Debug() bool {
	return *debug
}

// DBClient dbclient
func (fw *WMFrameWorkV2) DBClient() *db.SQLPool {
	return fw.dbCtl.client
}

// HTTPProtocol http协议
func (fw *WMFrameWorkV2) HTTPProtocol() string {
	return fw.httpProtocol
}
