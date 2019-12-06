package wlstmicro

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

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

// 数据库配置
type dbConfigure struct {
	// 数据库地址
	addr string
	// 登录用户名
	user string
	// 登录密码
	pwd string
	// 数据库名称
	database string
	// 数据库驱动模式，mssql/mysql
	driver string
	// 是否启用数据库
	enable bool
	// 是否启用tls
	usetls bool
}

// etcd配置
type etcdConfigure struct {
	// etcd服务地址
	addr string
	// 是否启用tls
	usetls bool
	// 是否启用etcd
	enable bool
	// 对外公布注册地址
	regAddr string
	// 注册根路径
	root string
}

// redis配置
type redisConfigure struct {
	// redis服务地址
	addr string
	// 访问密码
	pwd string
	// 数据库
	database int
	// 是否启用redis
	enable bool
}

// rabbitmq配置
type rabbitConfigure struct {
	// rmq服务地址
	addr string
	// 登录用户名
	user string
	// 登录密码
	pwd string
	// 虚拟域名
	vhost string
	// 交换机名称
	exchange string
	// 队列名称
	queue string
	// 是否使用随机队列名
	queueRandom bool
	// 队列是否持久化
	durable bool
	// 队列是否在未使用时自动删除
	autodel bool
	// 是否启用tls
	usetls bool
	// 是否启用rmq
	enable bool
	// 启用gps校时,0-不启用，1-启用（30～900s内进行矫正），2-强制对时
	gpsTiming int64
}

// 本地变量
var (
	StandAloneMode = gopsu.IsExist(".standalone")
	baseCAPath     string

	ETCDTLS    *tlsFiles
	HTTPTLS    *tlsFiles
	GRPCTLS    *tlsFiles
	RMQTLS     *tlsFiles
	AppConf    *gopsu.ConfData
	dbConf     = &dbConfigure{}
	redisConf  = &redisConfigure{}
	etcdConf   = &etcdConfigure{}
	rabbitConf = &rabbitConfigure{}

	rootPath = "wlst-micro"

	microLog *gopsu.MxLog

	MainPort   int
	LogLevel   int
	HTTPClient *http.Client
)

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

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// 创建固定目录
	gopsu.DefaultConfDir, gopsu.DefaultLogDir, gopsu.DefaultCacheDir = gopsu.MakeRuntimeDirs(".")
	// 配置默认ca文件路径
	baseCAPath = filepath.Join(gopsu.DefaultConfDir, "ca")
	// 检查是否有ca文件指向配置存在,存在则更新路径信息
	if a, err := ioutil.ReadFile(".capath"); err == nil {
		baseCAPath = gopsu.DecodeString(gopsu.TrimString(string(a)))
	}
	ETCDTLS = &tlsFiles{
		Cert:     filepath.Join(baseCAPath, "etcd.pem"),
		Key:      filepath.Join(baseCAPath, "etcd-key.pem"),
		ClientCA: filepath.Join(baseCAPath, "rootca.pem"),
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "etcd-ca.pem")) {
		ETCDTLS.ClientCA = filepath.Join(baseCAPath, "etcd-ca.pem")
	}
	HTTPTLS = &tlsFiles{
		Cert:     filepath.Join(baseCAPath, "http.pem"),
		Key:      filepath.Join(baseCAPath, "http-key.pem"),
		ClientCA: filepath.Join(baseCAPath, "rootca.pem"),
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "http-ca.pem")) {
		HTTPTLS.ClientCA = filepath.Join(baseCAPath, "http-ca.pem")
	}
	GRPCTLS = &tlsFiles{
		Cert:     filepath.Join(baseCAPath, "grpc.pem"),
		Key:      filepath.Join(baseCAPath, "grpc-key.pem"),
		ClientCA: filepath.Join(baseCAPath, "rootca.pem"),
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "grpc-ca.pem")) {
		GRPCTLS.ClientCA = filepath.Join(baseCAPath, "grpc-ca.pem")
	}
	RMQTLS = &tlsFiles{
		Cert:     filepath.Join(baseCAPath, "rmq.pem"),
		Key:      filepath.Join(baseCAPath, "rmq-key.pem"),
		ClientCA: filepath.Join(baseCAPath, "rootca.pem"),
	}
	if gopsu.IsExist(filepath.Join(baseCAPath, "rmq-ca.pem")) {
		RMQTLS.ClientCA = filepath.Join(baseCAPath, "rmq-ca.pem")
	}
	HTTPClient = &http.Client{
		Timeout: time.Duration(time.Second * 60),
		Transport: &http.Transport{
			IdleConnTimeout: time.Minute,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

// LoadConfigure 初始化配置
// f：配置文件路径，p：http端口，l：日志等级
// clientca：客户端ca路径（可选）
func LoadConfigure(f string, p, l int, clientca string) {
	if !strings.ContainsAny(f, "\\/") {
		f = filepath.Join(gopsu.DefaultConfDir, f)
	}
	AppConf, _ = gopsu.LoadConfig(f)
	rootPath = AppConf.GetItemDefault("root_path", "wlst-micro", "etcd/mq/redis注册根路径")
	rabbitConf.gpsTiming, _ = strconv.ParseInt(AppConf.GetItemDefault("mq_gpstiming", "0", "是否使用广播的gps时间进行对时操作,0-不启用，1-启用（30～900s内进行矫正），2-忽略误差范围强制矫正"), 10, 0)
	AppConf.Save()
	MainPort = p
	LogLevel = l
	if p > 0 && l > 0 {
		microLog = gopsu.NewLogger(gopsu.DefaultLogDir, "svr"+strconv.Itoa(p))
		microLog.SetLogLevel(l)
		if gopsu.IsExist(".synclog") {
			microLog.SetAsync(0)
		} else {
			microLog.SetAsync(1)
		}
	}
	HTTPTLS.ClientCA = clientca
	if rabbitConf.gpsTiming != 0 {
		go newGPSConsumer(strconv.Itoa(p))
	}
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
	if name == "" {
		if microLog == nil {
			println(fmt.Sprintf("%s", msg))
		} else {
			microLog.WriteLog(fmt.Sprintf("%s", msg), level)
		}
	} else {
		if microLog == nil {
			println(fmt.Sprintf("[%s] %s", name, msg))
		} else {
			microLog.WriteLog(fmt.Sprintf("[%s] %s", name, msg), level)
		}
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
