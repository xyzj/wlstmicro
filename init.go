package wlstmicro

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xyzj/gopsu"
)

type tlsFiles struct {
	Cert     string
	Key      string
	ClientCA string
}

type dbConfigure struct {
	addr     string
	user     string
	pwd      string
	database string
	driver   string
	enable   bool
	usetls   bool
}

type etcdConfigure struct {
	addr    string
	usetls  bool
	enable  bool
	regAddr string
	root    string
}

type redisConfigure struct {
	addr     string
	pwd      string
	database int
	enable   bool
}

type rabbitConfigure struct {
	addr     string
	user     string
	pwd      string
	vhost    string
	exchange string
	queue    string
	durable  bool
	autodel  bool
	usetls   bool
	enable   bool
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

	MainPort int
	LogLevel int
)

type stdLogger struct {
	Name string
}

// Debug Debug
func (l *stdLogger) Debug(msgs ...string) {
	WriteDebug(l.Name, strings.Join(msgs, ","))
}

// Info Info
func (l *stdLogger) Info(msgs ...string) {
	WriteInfo(l.Name, strings.Join(msgs, ","))
}

// Warn Warn
func (l *stdLogger) Warning(msgs ...string) {
	WriteWarning(l.Name, strings.Join(msgs, ","))
}

// Error Error
func (l *stdLogger) Error(msgs ...string) {
	WriteError(l.Name, strings.Join(msgs, ","))
}

// System System
func (l *stdLogger) System(msgs ...string) {
	WriteSystem(l.Name, strings.Join(msgs, ","))
}

func init() {
	gopsu.DefaultConfDir, gopsu.DefaultLogDir, gopsu.DefaultCacheDir = gopsu.MakeRuntimeDirs(".")
	if a, err := ioutil.ReadFile(".capath"); err == nil {
		baseCAPath = gopsu.DecodeString(string(a))
	}
	baseCAPath = filepath.Join(gopsu.DefaultConfDir, "ca")
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
}

// DefaultLogWriter 返回默认日志读写器
func DefaultLogWriter() io.Writer {
	return microLog.DefaultWriter()
}

// WriteDebug debug日志
func WriteDebug(name, msg string) {
	WriteLog(name, msg, 10)
}

// WriteInfo debug日志
func WriteInfo(name, msg string) {
	WriteLog(name, msg, 20)
}

// WriteWarning debug日志
func WriteWarning(name, msg string) {
	WriteLog(name, msg, 30)
}

// WriteError debug日志
func WriteError(name, msg string) {
	WriteLog(name, msg, 40)
}

// WriteSystem debug日志
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
