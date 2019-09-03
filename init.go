package wlstmicro

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/xyzj/gopsu"
)

type tlsFiles struct {
	cert     string
	key      string
	clientCA string
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
	usetls   bool
	enable   bool
}

// 本地变量
var (
	standAloneMode = gopsu.IsExist(".standalone")

	etcdTLS = &tlsFiles{
		cert:     filepath.Join(gopsu.DefaultConfDir, "ca", "etcd.pem"),
		key:      filepath.Join(gopsu.DefaultConfDir, "ca", "etcd-key.pem"),
		clientCA: filepath.Join(gopsu.DefaultConfDir, "ca", "rootca.pem"),
	}
	httpTLS = &tlsFiles{
		cert:     filepath.Join(gopsu.DefaultConfDir, "ca", "http.pem"),
		key:      filepath.Join(gopsu.DefaultConfDir, "ca", "http-key.pem"),
		clientCA: filepath.Join(gopsu.DefaultConfDir, "ca", "rootca.pem"),
	}
	grpcTLS = &tlsFiles{
		cert:     filepath.Join(gopsu.DefaultConfDir, "ca", "grpc.pem"),
		key:      filepath.Join(gopsu.DefaultConfDir, "ca", "grpc-key.pem"),
		clientCA: filepath.Join(gopsu.DefaultConfDir, "ca", "rootca.pem"),
	}

	appConf    *gopsu.ConfData
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

// InitConfigure 初始化配置
func InitConfigure(f string, p, l int) {
	appConf, _ = gopsu.LoadConfig(f)
	MainPort = p
	LogLevel = l
	sysLog = gopsu.NewLogger(gopsu.DefaultLogDir, "sys"+strconv.Itoa(p))
}

func writeLog(name, msg string, level int) {
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
