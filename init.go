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

	sysLog *gopsu.MxLog

	MainPort int
	LogLevel int
)

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
		sysLog = gopsu.NewLogger(gopsu.DefaultLogDir, "svr"+strconv.Itoa(p))
		sysLog.SetLogLevel(l)
		if gopsu.IsExist(".synclog") {
			sysLog.SetAsync(0)
		} else {
			sysLog.SetAsync(1)
		}
	}
	HTTPTLS.ClientCA = clientca
}

// DefaultLogWriter 返回默认日志读写器
func DefaultLogWriter() io.Writer {
	return sysLog.DefaultWriter
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
		if sysLog == nil {
			println(fmt.Sprintf("%s", msg))
		} else {
			sysLog.WriteLog(fmt.Sprintf("%s", msg), level)
		}
	} else {
		if sysLog == nil {
			println(fmt.Sprintf("[%s] %s", name, msg))
		} else {
			sysLog.WriteLog(fmt.Sprintf("[%s] %s", name, msg), level)
		}
	}
	// switch level {
	// case 10:
	// 	sysLog.Debug(fmt.Sprintf("[%s] %s", name, msg))
	// case 20:
	// 	sysLog.Info(fmt.Sprintf("[%s] %s", name, msg))
	// case 30:
	// 	sysLog.Warning(fmt.Sprintf("[%s] %s", name, msg))
	// case 40:
	// 	sysLog.Error(fmt.Sprintf("[%s] %s", name, msg))
	// default:
	// 	sysLog.System(fmt.Sprintf("[%s] %s", name, msg))
	// }
	// fmt.Fprintf(gin.DefaultWriter, "%s [%s] %s\n", time.Now().Format(logTimeFormat), name, msg)
	// if level > LogLevel && LogLevel > 10 {
	// 	fmt.Printf("%s [%s] %s\n", time.Now().Format(logTimeFormat), name, msg)
	// }
}

// rootPathMQ 返回MQ消息头,例 wlst-micro.
func rootPathMQ() string {
	return rootPath + "."
}

// rootPathRedis 返回redis key头,例 /wlst-micro/
func rootPathRedis() string {
	return "/" + rootPath + "/"
}
