package wlstmicro

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/xyzj/gopsu"
)

type tlsFiles struct {
	Cert     string
	Key      string
	ClientCA string
}

type mysqlConfigure struct {
	addr     string
	user     string
	pwd      string
	database string
	enable   bool
}

type etcdConfigure struct {
	addr    string
	usetls  bool
	enable  bool
	regAddr string
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

	ETCDTLS = &tlsFiles{
		Cert:     filepath.Join(gopsu.DefaultConfDir, "ca", "etcd.pem"),
		Key:      filepath.Join(gopsu.DefaultConfDir, "ca", "etcd-key.pem"),
		ClientCA: filepath.Join(gopsu.DefaultConfDir, "ca", "rootca.pem"),
	}
	HTTPTLS = &tlsFiles{
		Cert:     filepath.Join(gopsu.DefaultConfDir, "ca", "http.pem"),
		Key:      filepath.Join(gopsu.DefaultConfDir, "ca", "http-key.pem"),
		ClientCA: filepath.Join(gopsu.DefaultConfDir, "ca", "rootca.pem"),
	}
	GRPCTLS = &tlsFiles{
		Cert:     filepath.Join(gopsu.DefaultConfDir, "ca", "grpc.pem"),
		Key:      filepath.Join(gopsu.DefaultConfDir, "ca", "grpc-key.pem"),
		ClientCA: filepath.Join(gopsu.DefaultConfDir, "ca", "rootca.pem"),
	}

	AppConf    *gopsu.ConfData
	mysqlConf  = &mysqlConfigure{}
	redisConf  = &redisConfigure{}
	etcdConf   = &etcdConfigure{}
	rabbitConf = &rabbitConfigure{}

	sysLog *gopsu.MxLog

	MainPort int
	LogLevel int

	activeRedis bool
	activeMysql bool
	activeRmq   bool
	activeETCD  bool
)

// LoadConfigure 初始化配置
// f：配置文件路径，p：http端口，l：日志等级
// clientca：客户端ca路径（可选）
func LoadConfigure(f string, p, l int, clientca ...string) {
	AppConf, _ = gopsu.LoadConfig(f)
	MainPort = p
	LogLevel = l
	sysLog = gopsu.NewLogger(gopsu.DefaultLogDir, "sys"+strconv.Itoa(p))
	if len(clientca) > 0 {
		HTTPTLS.ClientCA = clientca[0]
	}
}

// WriteLog 写公共日志
func WriteLog(name, msg string, level int) {
	if sysLog == nil {
		println(fmt.Sprintf("[%s] %s", name, msg))
	} else {
		sysLog.WriteLog(fmt.Sprintf("[%s] %s", name, msg), level)
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
