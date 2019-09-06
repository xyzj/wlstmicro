package wlstmicro

import (
	"fmt"
	"strconv"
	"strings"

	"microgo"
)

var (
	etcdClient *microgo.Etcdv3Client
)

// NewETCDClient NewETCDClient
func NewETCDClient(svrName, svrType, svrProtocol string) {
	if AppConf == nil {
		WriteLog("SYS", "Configuration files should be loaded first", 40)
		return
	}
	etcdConf.addr = AppConf.GetItemDefault("etcd_addr", "127.0.0.1:2379", "etcd服务地址,ip:port格式")
	etcdConf.usetls, _ = strconv.ParseBool(AppConf.GetItemDefault("etcd_tls", "false", "是否使用证书连接etcd服务"))
	etcdConf.regAddr = AppConf.GetItemDefault("etcd_reg", "127.0.0.1", "服务注册地址,ip[:port]格式，不指定port时，自动使用http启动参数的端口")
	etcdConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("etcd_enable", "true", "是否启用etcd"))
	if etcdConf.usetls {
		etcdConf.addr = strings.Replace(etcdConf.addr, "2379", "2378", 1)
	}
	if !etcdConf.enable {
		return
	}
	var err error
	if etcdConf.usetls {
		etcdClient, err = microgo.NewEtcdv3ClientTLS([]string{etcdConf.addr}, ETCDTLS.Cert, ETCDTLS.Key, ETCDTLS.ClientCA)
	} else {
		etcdClient, err = microgo.NewEtcdv3Client([]string{etcdConf.addr})
	}
	if err != nil {
		WriteLog("ETCD", "Failed connect to "+etcdConf.addr+"|"+err.Error(), 40)
		activeETCD = false
		return
	}
	activeETCD = true
	etcdClient.SetLogger(&sysLog.DefaultWriter, LogLevel)

	// 注册自身
	a := strings.Split(etcdConf.regAddr, ":")
	regPort := strconv.Itoa(MainPort)
	if len(a) > 1 {
		regPort = a[1]
	}
	etcdClient.Register(svrName, a[0], regPort, svrType, svrProtocol)
	// etcdClient.Register("usermanager", a[0], regPort, "http", "json")
	// 获取服务列表信息
	etcdClient.Watcher()
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
