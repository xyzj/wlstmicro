package wlstmicro

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/microgo"
)

var (
	etcdClient *microgo.Etcdv3Client
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
	// 对外公布注册地址
	regAddr string
	// 注册根路径
	root string
}

func (conf *etcdConfigure) show() string {
	conf.forshow, _ = sjson.Set("", "addr", etcdConf.addr)
	conf.forshow, _ = sjson.Set(conf.forshow, "root", etcdConf.root)
	return conf.forshow
}

// NewETCDClient NewETCDClient
func NewETCDClient(svrName, svrType, svrProtocol string) bool {
	if AppConf == nil {
		WriteError("SYS", "Configuration files should be loaded first")
		return false
	}
	etcdConf.addr = AppConf.GetItemDefault("etcd_addr", "127.0.0.1:2379", "etcd服务地址,ip:port格式")
	etcdConf.regAddr = AppConf.GetItemDefault("etcd_reg", "127.0.0.1", "服务注册地址,ip[:port]格式，不指定port时，自动使用http启动参数的端口")
	etcdConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("etcd_enable", "true", "是否启用etcd"))
	etcdConf.usetls, _ = strconv.ParseBool(AppConf.GetItemDefault("etcd_tls", "false", "是否使用证书连接etcd服务"))
	if etcdConf.usetls {
		etcdConf.addr = strings.Replace(etcdConf.addr, "2379", "2378", 1)
	}
	if etcdConf.regAddr == "127.0.0.1" || etcdConf.regAddr == "" {
		etcdConf.regAddr, _ = gopsu.ExternalIP()
		AppConf.UpdateItem("etcd_reg", etcdConf.regAddr)
	}
	AppConf.Save()
	etcdConf.show()
	if !etcdConf.enable {
		return false
	}
	var err error
	if etcdConf.usetls {
		etcdClient, err = microgo.NewEtcdv3ClientTLS([]string{etcdConf.addr}, ETCDTLS.Cert, ETCDTLS.Key, ETCDTLS.ClientCA)
	} else {
		etcdClient, err = microgo.NewEtcdv3Client([]string{etcdConf.addr})
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
	a := strings.Split(etcdConf.regAddr, ":")
	regPort := strconv.Itoa(*WebPort)
	if len(a) > 1 {
		regPort = a[1]
	}
	etcdClient.Register(svrName, a[0], regPort, svrType, svrProtocol)
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
