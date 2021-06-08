package wmv2

import (
	"flag"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/xyzj/gopsu"
)

// 启动参数
var (
	// pyroscope debug
	pyroscope = flag.Bool("pyroscope", false, "set true to enable pyroscope debug, should only be used in DEV-LAN")
	// forceHTTP 强制http
	forceHTTP = flag.Bool("forcehttp", false, "set true to use HTTP anyway.")
	//  是否启用调试模式
	debug = flag.Bool("debug", false, "set if enable debug info.")
	// logLevel 日志等级，可选项10,20,30,40
	logLevel = flag.Int("loglevel", 20, "set the file log level. Enable value is: 10,20,30,40; 0-disable file log; -1-disable all log")
	// logDays 日志文件保留天数，默认15
	logDays = flag.Int("logdays", 10, "set the max days of the log files to keep")
	// webPort 主端口
	webPort = flag.Int("http", 6819, "set http port to listen on.")
	// ca文件夹路径
	capath = flag.String("capath", "", "set the ca files path")
	// portable 把日志，缓存等目录创建在当前目录下，方便打包带走
	portable = flag.Bool("portable", false, "把日志，配置，缓存目录创建在当前目录下")
	// 配置文件
	conf = flag.String("conf", "", "set the config file path.")
	// 版本信息
	ver = flag.Bool("version", false, "print version info and exit.")
	// 帮助信息
	help = flag.Bool("help", false, "print help message and exit.")
)

var (
	// CWorker 加密
	CWorker *gopsu.CryptoWorker // = gopsu.GetNewCryptoWorker(gopsu.CryptoAES128CBC)
	// MD5Worker md5计算
	MD5Worker *gopsu.CryptoWorker // = gopsu.GetNewCryptoWorker(gopsu.CryptoMD5)
)

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
	// 启动分表线程
	DoMERGE bool
	// 启用
	Activation bool
	// 设置升级脚本
	DBUpgrade []byte
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

// OptionMQGPSTimer rmq gps timer 配置
type OptionMQGPSTimer struct {
	// 启用
	Activation bool
}

// OptionMQConsumer rmq配置
type OptionMQConsumer struct {
	// 消费者绑定key
	BindKeys []string
	// 消费者key获取方法
	BindKeysFunc func() ([]string, bool)
	// 消费者数据处理方法
	RecvFunc func(key string, body []byte)
	// 启用
	Activation bool
}

// OptionHTTP http配置
type OptionHTTP struct {
	// 路由引擎组合方法，推荐使用这个方法代替GinEngine值，可以避免过早初始化
	EngineFunc func() *gin.Engine
	// 启用
	Activation bool
}

// OptionTCP tcp配置
type OptionTCP struct {
	// 启用
	Activation bool
	// 端口
	BindPort int
	// mqflag
	MQFlag string
	// client 接口实现
	Client TCPBase
}

// OptionFrameWorkV2 wlst 微服务框架配置v2版
type OptionFrameWorkV2 struct {
	// 配置文件路径
	ConfigFile string
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
	// 启用tcp服务模块
	UseTCP *OptionTCP
	// 启动参数处理方法，在功能模块初始化之前执行
	// 提交方法名称时最后不要加`()`，表示把方法作为参数，而不是把方法的执行结果回传
	FrontFunc func()
	// 扩展方法列表，用于处理额外的数据或变量，在主要模块启动完成后依次执行
	// 非线程顺序执行，注意不要阻塞
	// sample：
	// []func(){
	//	 FuncA,
	//	 go FuncB
	// }
	ExpandFuncs []func()
}

// WMFrameWorkV2 v2版微服务框架
type WMFrameWorkV2 struct {
	// 变量类
	serverName    string
	loggerMark    string
	verJSON       string
	tag           string
	startAt       string
	tokenLife     time.Duration
	rootPath      string
	rootPathRedis string
	rootPathMQ    string
	gpsTimer      int64 // 启用gps校时,0-不启用，1-启用（30～900s内进行矫正），2-强制对时
	httpProtocol  string
	// tls配置
	baseCAPath string
	tlsCert    string //  = filepath.Join(baseCAPath, "client-cert.pem")
	tlsKey     string //  = filepath.Join(baseCAPath, "client-key.pem")
	tlsRoot    string //  = filepath.Join(baseCAPath, "rootca.pem")
	httpCert   string
	httpKey    string

	// 配置
	wmConf *gopsu.ConfData
	// 日志
	wmLog gopsu.Logger
	// 模块实例类
	chanTCPWorker  chan interface{}
	etcdCtl        *etcdConfigure
	redisCtl       *redisConfigure
	dbCtl          *dbConfigure
	rmqCtl         *rabbitConfigure
	tcpCtl         *tcpConfigure
	httpClientPool *http.Client
	JSON           jsoniter.API
	cnf            *OptionFrameWorkV2
}

func init() {
	// 设置使用的cpu核心数量
	runtime.GOMAXPROCS(runtime.NumCPU())
	// CWorker 加密
	CWorker = gopsu.GetNewCryptoWorker(gopsu.CryptoAES128CBC)
	CWorker.SetKey("(NMNle+XW!ykVjf1", "Zq0V+,.2u|3sGAzH")
	// MD5Worker md5计算
	MD5Worker = gopsu.GetNewCryptoWorker(gopsu.CryptoMD5)
}
