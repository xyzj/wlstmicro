package wlstmicro

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/microgo"
)

var (
	etcdClient *microgo.Etcdv3Client
	etcdConf   = &etcdConfigure{}
)

// etcd配置
type etcdConfigure struct {
	forshow string
	// etcd服务地址
	addr string
	// 是否启用tls
	usetls bool
	// 是否启用etcd
	enable bool
	// 是否优先使用v6地址
	v6 bool
	// 对外公布注册地址
	regAddr string
	// 注册根路径
	root string
	// enable auth
	useauth bool
	// user
	username string
	// passwd
	password string
}

func (conf *etcdConfigure) show() string {
	conf.forshow, _ = sjson.Set("", "addr", etcdConf.addr)
	conf.forshow, _ = sjson.Set(conf.forshow, "use_tls", etcdConf.usetls)
	conf.forshow, _ = sjson.Set(conf.forshow, "root_path", rootPath)
	return conf.forshow
}

// NewETCDClient NewETCDClient
func NewETCDClient(svrName, svrType, svrProtocol string) bool {
	serverName = svrName
	if AppConf == nil {
		WriteError("SYS", "Configuration files should be loaded first")
		return false
	}
	etcdConf.addr = AppConf.GetItemDefault("etcd_addr", "127.0.0.1:2378", "etcd服务地址,ip:port格式")
	etcdConf.regAddr = AppConf.GetItemDefault("etcd_reg", "127.0.0.1", "服务注册地址,ip[:port]格式，不指定port时，自动使用http启动参数的端口")
	etcdConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("etcd_enable", "true", "是否启用etcd"))
	etcdConf.useauth, _ = strconv.ParseBool(AppConf.GetItemDefault("etcd_auth", "true", "连接etcd时是否需要认证"))
	etcdConf.usetls, _ = strconv.ParseBool(AppConf.GetItemDefault("etcd_tls", "true", "是否使用证书连接etcd服务"))
	etcdConf.v6, _ = strconv.ParseBool(AppConf.GetItemDefault("etcd_v6", "false", "是否优先使用v6地址"))
	if !etcdConf.usetls {
		etcdConf.addr = strings.Replace(etcdConf.addr, "2378", "2379", 1)
	}
	if etcdConf.regAddr == "127.0.0.1" || etcdConf.regAddr == "" {
		etcdConf.regAddr = gopsu.RealIP(etcdConf.v6)
		AppConf.UpdateItem("etcd_reg", etcdConf.regAddr)
	}
	AppConf.Save()
	etcdConf.show()
	if !etcdConf.enable {
		return false
	}
	if etcdConf.useauth {
		etcdConf.username = "root"
		etcdConf.password = gopsu.DecodeString("wMQLEoOHM2eOF6O7Ho8MH74jZ1vMs5i1B+VL+ozl")
	}
	var err error
	if etcdConf.usetls {
		etcdClient, err = microgo.NewEtcdv3ClientTLS([]string{etcdConf.addr}, ETCDTLS.Cert, ETCDTLS.Key, ETCDTLS.ClientCA, etcdConf.username, etcdConf.password)
	} else {
		etcdClient, err = microgo.NewEtcdv3Client([]string{etcdConf.addr}, etcdConf.username, etcdConf.password)
	}
	if err != nil {
		WriteError("ETCD", "Failed connect to "+etcdConf.addr+"|"+err.Error())
		return false
	}
	etcdClient.SetLogger(&StdLogger{
		Name:        "ETCD",
		LogReplacer: strings.NewReplacer("[", "", "]", ""),
	})

	// 注册自身
	if len(rootPath) > 0 {
		etcdClient.SetRoot(rootPath)
	}
	a, b, err := net.SplitHostPort(etcdConf.regAddr)
	if err != nil {
		a = etcdConf.regAddr
	}
	if b == "" {
		b = strconv.Itoa(*WebPort)
	}
	etcdClient.Register(svrName, a, b, svrType, svrProtocol)
	// 获取服务列表信息
	etcdClient.Watcher()
	return true
}

// ETCDIsReady 返回ETCD可用状态
func ETCDIsReady() bool {
	return etcdClient != nil
}

// Picker 选取服务地址
func Picker(svrName string) (string, error) {
	if etcdClient == nil {
		return "", fmt.Errorf("etcd client not ready")
	}
	addr, err := etcdClient.Picker(svrName)
	if err != nil {
		return "", err
	}
	return addr, nil
}

// PickerDetail 选取服务地址,带http(s)前缀
func PickerDetail(svrName string) (string, error) {
	if etcdClient == nil {
		return "", fmt.Errorf("etcd client not ready")
	}
	addr, err := etcdClient.PickerDetail(svrName)
	if err != nil {
		return "", err
	}
	return addr, nil
}

// ViewETCDConfig 查看ETCD配置,返回json字符串
func ViewETCDConfig() string {
	return etcdConf.forshow
}
