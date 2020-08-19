package wlstmicro

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	ginmiddleware "github.com/xyzj/gopsu/gin-middleware"

	"github.com/gin-gonic/gin"
	"github.com/xyzj/gopsu"
)

// tls配置
type tlsFiles struct {
	// ca证书
	Cert string
	// ca key
	Key string
	// 客户端ca根证书
	ClientCA string
}

// 本地变量
var (
	baseCAPath string

	ETCDTLS *tlsFiles
	HTTPTLS *tlsFiles
	GRPCTLS *tlsFiles
	RMQTLS  *tlsFiles
	AppConf *gopsu.ConfData
	//	根路径
	rootPath = "wlst-micro"
	// 日志
	microLog gopsu.Logger
	// 域名
	domainName = ""
	// HTTPClient http request 池
	HTTPClient *http.Client

	// 加密解密worker
	CWorker   = gopsu.GetNewCryptoWorker(gopsu.CryptoAES128CBC)
	MD5Worker = gopsu.GetNewCryptoWorker(gopsu.CryptoMD5)
	// Token 时效
	tokenLife = time.Minute * 30
	// 日志标识
	loggerMark  string
	VersionInfo string
	serverName  string
	// http服务的协议，http or https
	HTTPProtocol string
)

// 启动参数
var (
	// forceHTTP 强制http
	forceHTTP = flag.Bool("forcehttp", false, "set true to use HTTP anyway.")
	// Debug 是否启用调试模式
	Debug = flag.Bool("debug", false, "set if enable debug info.")
	// logLevel 日志等级，可选项10,20,30,40
	logLevel = flag.Int("loglevel", 20, "set the file log level. Enable value is: 10,20,30,40; 0-disable file log; -1-disable all log")
	// logDays 日志文件保留天数，默认15
	logDays = flag.Int("logdays", 15, "set the max days of the log files to keep")
	// WebPort 主端口
	WebPort = flag.Int("http", 6819, "set http port to listen on.")
	// portable 把日志，缓存等目录创建在当前目录下，方便打包带走
	portable = flag.Bool("portable", false, "把日志，配置，缓存目录创建在当前目录下")
	// 配置文件
	conf = flag.String("conf", "", "set the config file path.")
	// 版本信息
	ver = flag.Bool("version", false, "print version info and exit.")
	// 帮助信息
	help = flag.Bool("help", false, "print help message and exit.")
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	microLog = &gopsu.StdLogger{}

	CWorker.SetKey("(NMNle+XW!ykVjf1", "Zq0V+,.2u|3sGAzH")
	// 创建固定目录
	// gopsu.DefaultConfDir, gopsu.DefaultLogDir, gopsu.DefaultCacheDir = gopsu.MakeRuntimeDirs("..")
	// 配置默认ca文件路径
	baseCAPath = filepath.Join(gopsu.DefaultConfDir, "ca")
	// 检查是否有ca文件指向配置存在,存在则更新路径信息
	if a, err := ioutil.ReadFile(".capath"); err == nil {
		baseCAPath = gopsu.DecodeString(gopsu.TrimString(string(a)))
	}
	ETCDTLS = &tlsFiles{
		Cert:     filepath.Join(baseCAPath, "client-cert.pem"),
		Key:      filepath.Join(baseCAPath, "client-key.pem"),
		ClientCA: filepath.Join(baseCAPath, "rootca.pem"),
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "etcd-ca.pem")) {
		ETCDTLS.ClientCA = filepath.Join(baseCAPath, "etcd-ca.pem")
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "etcd-cert.pem")) {
		ETCDTLS.Cert = filepath.Join(baseCAPath, "etcd-cert.pem")
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "etcd-key.pem")) {
		ETCDTLS.Key = filepath.Join(baseCAPath, "etcd-key.pem")
	}
	HTTPTLS = &tlsFiles{
		Cert: filepath.Join(baseCAPath, "client-cert.pem"),
		Key:  filepath.Join(baseCAPath, "client-key.pem"),
	}
	GRPCTLS = &tlsFiles{
		Cert:     filepath.Join(baseCAPath, "client-cert.pem"),
		Key:      filepath.Join(baseCAPath, "client-key.pem"),
		ClientCA: filepath.Join(baseCAPath, "rootca.pem"),
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "grpc-ca.pem")) {
		GRPCTLS.ClientCA = filepath.Join(baseCAPath, "grpc-ca.pem")
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "grpc-cert.pem")) {
		GRPCTLS.Cert = filepath.Join(baseCAPath, "grpc-cert.pem")
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "grpc-key.pem")) {
		GRPCTLS.Key = filepath.Join(baseCAPath, "grpc-key.pem")
	}
	RMQTLS = &tlsFiles{
		Cert:     filepath.Join(baseCAPath, "client-cert.pem"),
		Key:      filepath.Join(baseCAPath, "client-key.pem"),
		ClientCA: filepath.Join(baseCAPath, "rootca.pem"),
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "rmq-ca.pem")) {
		RMQTLS.ClientCA = filepath.Join(baseCAPath, "rmq-ca.pem")
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "rmq-cert.pem")) {
		RMQTLS.Cert = filepath.Join(baseCAPath, "rmq-cert.pem")
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "rmq-key.pem")) {
		RMQTLS.Key = filepath.Join(baseCAPath, "rmq-key.pem")
	}
}

// OptionETCD ETCD配置
type OptionETCD struct {
	// 服务名称
	SvrName string
	// 服务类型，留空时默认为http或https
	SvrType string
	// 交互协议，留空默认json
	SvrProtocol string
	// 启用
	Activation bool
}

// OptionSQL 数据库配置
type OptionSQL struct {
	// 缓存文件标识
	CacheMark string
	// 启动分表线程
	DoMERGE bool
	// 启用
	Activation bool
}

// OptionRedis redis配置
type OptionRedis struct {
	// 启用
	Activation bool
}

// OptionMQProducer rmq配置
type OptionMQProducer struct {
	// 启用
	Activation bool
}

// OptionMQConsumer rmq配置
type OptionMQConsumer struct {
	// 消费者队列名
	QueueName string
	// 消费者绑定key
	BindKeys []string
	// 消费者数据处理方法
	RecvFunc func(key string, body []byte)
	// 启用
	Activation bool
}

// OptionHTTP http配置
type OptionHTTP struct {
	// 路由引擎值
	GinEngine *gin.Engine
	// 路由引擎组合方法，推荐使用这个方法代替GinEngine值，可以避免过早初始化
	EngineFunc func() *gin.Engine
	// 启用
	Activation bool
	// 启用apirecord功能
	RecordAPI bool
}

// ExpandFunc 扩展带参数方法
type ExpandFunc struct {
	Func func(args ...interface{})
	Args []interface{}
}

// OptionFramework go语言微服务框架
type OptionFramework struct {
	LoggerMark string
	Version    string
	// 启用ETCD模块
	UseETCD *OptionETCD
	// 启用SQL模块
	UseSQL *OptionSQL
	// 启用Redis模块
	UseRedis *OptionRedis
	// 启用mq生产者模块
	UseMQProducer *OptionMQProducer
	// 启用mq消费者模块
	UseMQConsumer *OptionMQConsumer
	// 启用http服务模块
	UseHTTP *OptionHTTP
	// 启动参数处理方法，在模块初始化之前执行
	// 非线程执行，注意不要阻塞
	// 提交方法名称时最后不要加`()`，表示把方法作为参数，而不是把方法的执行结果回传
	FrontFunc func()
	// 无参数的扩展方法，用于处理额外的数据或变量，所有模块初始化完成后执行
	// 非线程执行，注意不要阻塞
	// 提交方法名称时最后不要加`()`，表示把方法作为参数，而不是把方法的执行结果回传
	ExpandFunc func()
	// 带参数的扩展方法列表，，用于处理额外的数据或变量，可传入外部参数，ExpandFunc执行完成后执行
	// 非线程执行，注意不要阻塞
	// 非线程顺序执行，注意不要阻塞
	// sample：
	// []*wlstmicro.ExpandFunc{
	// 	&wlstmicro.ExpandFunc{
	// 		Func: funcName,
	// 		Args: []interface{}{"aaa",111},
	// 	},
	// }
	// ExpandFuncsWithArgs []*ExpandFunc
	ExpandFuncs []func()
}

// getFlagReady 处理启动参数
func getFlagReady() {
	if !flag.Parsed() {
		flag.Parse()
	}
	if *help {
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *ver {
		println(VersionInfo)
		os.Exit(1)
	}
}

// RunFramework 初始化框架相关参数
func RunFramework(om *OptionFramework) {
	// 保存版本信息
	if om.Version != "" {
		VersionInfo = om.Version
		p, _ := os.Executable()
		f, _ := os.OpenFile(fmt.Sprintf("%s.ver", p), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0444)
		defer f.Close()
		f.WriteString(VersionInfo + "\r\n")
	}
	// 处置参数
	getFlagReady()
	// 处置目录
	if *portable {
		gopsu.DefaultConfDir, gopsu.DefaultLogDir, gopsu.DefaultCacheDir = gopsu.MakeRuntimeDirs(".")
	} else {
		gopsu.DefaultConfDir, gopsu.DefaultLogDir, gopsu.DefaultCacheDir = gopsu.MakeRuntimeDirs("..")
	}
	// 前置处理方法，用于预初始化某些内容
	if om.FrontFunc != nil {
		om.FrontFunc()
	}
	if om.LoggerMark == "" {
		if om.UseETCD == nil {
			loggerMark = fmt.Sprintf("X-%05d", *WebPort)
		} else {
			loggerMark = fmt.Sprintf("%s-%05d", om.UseETCD.SvrName, *WebPort)
		}
	} else {
		loggerMark = om.LoggerMark
	}
	if *Debug {
		*logLevel = 10
	}
	switch *logLevel {
	case -1:
		microLog = &gopsu.NilLogger{}
	case 0:
		microLog = &gopsu.StdLogger{}
	default:
		microLog = gopsu.NewLogger(gopsu.DefaultLogDir, loggerMark+".core", *logLevel, *logDays)
	}
	// 载入配置
	LoadConfigure()
	// 逐步启动服务
	if om.UseETCD != nil {
		if om.UseETCD.Activation {
			if om.UseETCD.SvrType == "" {
				if *Debug || *forceHTTP {
					om.UseETCD.SvrType = "http"
				} else {
					om.UseETCD.SvrType = "https"
				}
			}
			if om.UseETCD.SvrProtocol == "" {
				om.UseETCD.SvrProtocol = "json"
			}
			NewETCDClient(om.UseETCD.SvrName, om.UseETCD.SvrType, om.UseETCD.SvrProtocol)
		}
	}
	if om.UseRedis != nil {
		if om.UseRedis.Activation {
			NewRedisClient()
		}
	}
	if om.UseSQL != nil {
		if om.UseSQL.Activation {
			if om.UseSQL.CacheMark == "" {
				om.UseSQL.CacheMark = strconv.FormatInt(int64(*WebPort), 10)
			}
			if NewMysqlClient(om.UseSQL.CacheMark) {
				if om.UseSQL.DoMERGE {
					go MaintainMrgTables()
				}
			}
		}
	}
	if om.UseHTTP != nil {
		if om.UseHTTP.Activation {
			yaagEnable = om.UseHTTP.RecordAPI
			if om.UseHTTP.EngineFunc != nil {
				om.UseHTTP.GinEngine = om.UseHTTP.EngineFunc()
			} else {
				if om.UseHTTP.GinEngine == nil {
					om.UseHTTP.GinEngine = NewHTTPEngine()
				}
			}
			NewHTTPService(om.UseHTTP.GinEngine)
			if VersionInfo != "" {
				ginmiddleware.SetVersionInfo(VersionInfo)
			}
		}
	}
	if om.UseMQProducer != nil {
		if om.UseMQProducer.Activation {
			NewMQProducer()
		}
	}
	if om.UseMQConsumer != nil {
		if om.UseMQConsumer.Activation {
			if NewMQConsumer(om.UseMQConsumer.QueueName) {
				BindRabbitMQ(om.UseMQConsumer.BindKeys...)
				go RecvRabbitMQ(om.UseMQConsumer.RecvFunc)
			}
		}
	}
	// 执行gps对时
	if rabbitConf != nil {
		if rabbitConf.gpsTiming != 0 {
			go newGPSConsumer(strconv.Itoa(*WebPort))
		}
	}
	// 执行额外方法
	if om.ExpandFunc != nil {
		om.ExpandFunc()
	}
	if om.ExpandFuncs != nil {
		for _, v := range om.ExpandFuncs {
			v()
		}
	}
	for {
		time.Sleep(time.Minute)
	}
}

// LoadConfigure 初始化配置
// 以下可选，
// f：配置文件路径，
// p：日志文件标识，默认使用主端口号，为0,不启用日志，l：日志等级
// clientca：客户端ca路径(作废，改为配置文件指定)
func LoadConfigure() {
	// 检查配置
	if *conf == "" {
		println("no config file set")
		os.Exit(1)
	}
	f := *conf
	if !strings.ContainsAny(f, "\\/") {
		f = filepath.Join(gopsu.DefaultConfDir, f)
	}
	if AppConf == nil {
		AppConf, _ = gopsu.LoadConfig(f)
	}
	rootPath = AppConf.GetItemDefault("root_path", "wlst-micro", "etcd/mq/redis注册根路径")
	domainName = AppConf.GetItemDefault("domain_name", "", "set the domain name, cert and key file name should be xxx.crt & xxx.key")
	rabbitConf.gpsTiming, _ = strconv.ParseInt(AppConf.GetItemDefault("mq_gpstiming", "0", "是否使用广播的gps时间进行对时操作,0-不启用，1-启用（30～900s内进行矫正），2-忽略误差范围强制矫正"), 10, 0)
	HTTPTLS.ClientCA = AppConf.GetItemDefault("client_ca", "", "双向认证用ca文件路径")
	AppConf.Save()
	if domainName != "" {
		HTTPTLS.Cert = filepath.Join(baseCAPath, domainName+".crt")
		HTTPTLS.Key = filepath.Join(baseCAPath, domainName+".key")
	}
	// 以下三个参数不自动生成，影响dorequest性能
	// request超时时间（秒）
	var trTimeo = time.Second * 60
	// 最大idle连接保持数量
	var trMaxidle = 0
	// 每个host允许的最大连接数
	var trMaxconnPerHost = 10
	s, err := AppConf.GetItem("tr_timeo")
	if err == nil {
		if gopsu.String2Int(s, 10) > 2 {
			trTimeo = time.Second * time.Duration(gopsu.String2Int(s, 10))
		}
	}
	s, err = AppConf.GetItem("tr_maxidle")
	if err == nil {
		trMaxidle = gopsu.String2Int(s, 10)
	}
	s, err = AppConf.GetItem("tr_maxconn_perhost")
	if err == nil {
		trMaxconnPerHost = gopsu.String2Int(s, 10)
	}
	HTTPClient = &http.Client{
		Timeout: time.Duration(trTimeo),
		Transport: &http.Transport{
			IdleConnTimeout:     time.Second * 10,
			MaxConnsPerHost:     trMaxconnPerHost,
			MaxIdleConns:        trMaxidle,
			MaxIdleConnsPerHost: 1,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

// SetTokenLife 设置User-Token的有效期，默认30分钟
func SetTokenLife(t time.Duration) {
	tokenLife = t
}

// StdLogger StdLogger
type StdLogger struct {
	Name        string
	LogReplacer *strings.Replacer
}

// Debug Debug
func (l *StdLogger) Debug(msgs string) {
	WriteDebug(l.Name, msgs)
}

// Info Info
func (l *StdLogger) Info(msgs string) {
	WriteInfo(l.Name, msgs)
}

// Warning Warn
func (l *StdLogger) Warning(msgs string) {
	WriteWarning(l.Name, msgs)
}

// Error Error
func (l *StdLogger) Error(msgs string) {
	WriteError(l.Name, msgs)
}

// System System
func (l *StdLogger) System(msgs string) {
	WriteSystem(l.Name, msgs)
}

// DebugFormat Debug
func (l *StdLogger) DebugFormat(f string, msg ...interface{}) {
	if f == "" {
		WriteDebug(l.Name, l.LogReplacer.Replace(fmt.Sprintf("%v", msg)))
	} else {
		WriteDebug(l.Name, fmt.Sprintf(f, msg...))
	}
}

// InfoFormat Info
func (l *StdLogger) InfoFormat(f string, msg ...interface{}) {
	if f == "" {
		WriteInfo(l.Name, l.LogReplacer.Replace(fmt.Sprintf("%v", msg)))
	} else {
		WriteInfo(l.Name, fmt.Sprintf(f, msg...))
	}
}

// WarningFormat Warn
func (l *StdLogger) WarningFormat(f string, msg ...interface{}) {
	if f == "" {
		WriteWarning(l.Name, l.LogReplacer.Replace(fmt.Sprintf("%v", msg)))
	} else {
		WriteWarning(l.Name, fmt.Sprintf(f, msg...))
	}
}

// ErrorFormat Error
func (l *StdLogger) ErrorFormat(f string, msg ...interface{}) {
	if f == "" {
		WriteError(l.Name, l.LogReplacer.Replace(fmt.Sprintf("%v", msg)))
	} else {
		WriteError(l.Name, fmt.Sprintf(f, msg...))
	}
}

// SystemFormat System
func (l *StdLogger) SystemFormat(f string, msg ...interface{}) {
	if f == "" {
		WriteSystem(l.Name, l.LogReplacer.Replace(fmt.Sprintf("%v", msg)))
	} else {
		WriteSystem(l.Name, fmt.Sprintf(f, msg...))
	}
}

// DefaultWriter 返回默认writer
func (l *StdLogger) DefaultWriter() io.Writer {
	return microLog.DefaultWriter()
}

// DefaultLogWriter 返回默认日志读写器
func DefaultLogWriter() io.Writer {
	return microLog.DefaultWriter()
}

// WriteDebug debug日志
func WriteDebug(name, msg string) {
	WriteLog(name, msg, 10)
}

// WriteInfo Info日志
func WriteInfo(name, msg string) {
	WriteLog(name, msg, 20)
}

// WriteWarning Warning日志
func WriteWarning(name, msg string) {
	WriteLog(name, msg, 30)
}

// WriteError Error日志
func WriteError(name, msg string) {
	WriteLog(name, msg, 40)
}

// WriteSystem System日志
func WriteSystem(name, msg string) {
	WriteLog(name, msg, 90)
}

// WriteLog 写公共日志
// name： 日志类别，如sys，mq，db这种
// msg： 日志信息
// level： 日志级别10,20，30,40,90
func WriteLog(name, msg string, level int) {
	if level == -1 || level < *logLevel {
		return
	}
	if name != "" {
		name = "[" + name + "] "
	}
	msg = gopsu.TrimString(msg)
	switch level {
	case 10:
		microLog.Debug(fmt.Sprintf("%s%s", name, msg))
	case 20:
		microLog.Info(fmt.Sprintf("%s%s", name, msg))
	case 30:
		microLog.Warning(fmt.Sprintf("%s%s", name, msg))
	case 40:
		microLog.Error(fmt.Sprintf("%s%s", name, msg))
	case 90:
		microLog.System(fmt.Sprintf("%s%s", name, msg))
	}
}

// rootPathMQ 返回MQ消息头,例 wlst-micro.
func rootPathMQ() string {
	return rootPath + "."
}

// rootPathRedis 返回redis key头,例 /wlst-micro/
func rootPathRedis() string {
	return "/" + rootPath + "/"
}
