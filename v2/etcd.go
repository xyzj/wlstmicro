package wmv2

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/microgo"
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
	// 优先v6
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
	// Client
	Client *microgo.Etcdv3Client
}

func (conf *etcdConfigure) show(rootPath string) string {
	conf.forshow, _ = sjson.Set("", "addr", conf.addr)
	conf.forshow, _ = sjson.Set(conf.forshow, "use_tls", conf.usetls)
	conf.forshow, _ = sjson.Set(conf.forshow, "root_path", rootPath)
	return conf.forshow
}

// NewETCDClient NewETCDClient
func (fw *WMFrameWorkV2) newETCDClient() bool {
	fw.etcdCtl.addr = fw.wmConf.GetItemDefault("etcd_addr", "127.0.0.1:2378", "etcd服务地址,ip:port格式")
	fw.etcdCtl.regAddr = fw.wmConf.GetItemDefault("etcd_reg", "", "服务注册地址,ip[:port]格式，不指定port时，自动使用http启动参数的端口")
	fw.etcdCtl.enable, _ = strconv.ParseBool(fw.wmConf.GetItemDefault("etcd_enable", "true", "是否启用etcd"))
	fw.etcdCtl.useauth, _ = strconv.ParseBool(fw.wmConf.GetItemDefault("etcd_auth", "true", "连接etcd时是否需要认证"))
	fw.etcdCtl.usetls, _ = strconv.ParseBool(fw.wmConf.GetItemDefault("etcd_tls", "true", "是否使用证书连接etcd服务"))
	fw.etcdCtl.v6, _ = strconv.ParseBool(fw.wmConf.GetItemDefault("etcd_v6", "false", "是否优先使用v6地址"))
	if !fw.etcdCtl.usetls {
		fw.etcdCtl.addr = strings.Replace(fw.etcdCtl.addr, "2378", "2379", 1)
	}
	if fw.etcdCtl.regAddr == "127.0.0.1" || fw.etcdCtl.regAddr == "" {
		fw.etcdCtl.regAddr = gopsu.RealIP(fw.etcdCtl.v6)
		fw.wmConf.UpdateItem("etcd_reg", fw.etcdCtl.regAddr)
	}
	fw.wmConf.Save()
	fw.etcdCtl.show(fw.rootPath)
	if !fw.etcdCtl.enable {
		return false
	}
	if fw.etcdCtl.useauth {
		fw.etcdCtl.username = "root"
		fw.etcdCtl.password = gopsu.DecodeString("wMQLEoOHM2eOF6O7Ho8MH74jZ1vMs5i1B+VL+ozl")
	}
	var err error
	if fw.etcdCtl.usetls {
		fw.etcdCtl.Client, err = microgo.NewEtcdv3ClientTLS([]string{fw.etcdCtl.addr}, fw.tlsCert, fw.tlsKey, fw.tlsRoot, fw.etcdCtl.username, fw.etcdCtl.password)
	} else {
		fw.etcdCtl.Client, err = microgo.NewEtcdv3Client([]string{fw.etcdCtl.addr}, fw.etcdCtl.username, fw.etcdCtl.password)
	}
	if err != nil {
		fw.etcdCtl.enable = false
		fw.WriteError("ETCD", "Failed connect to "+fw.etcdCtl.addr+"|"+err.Error())
		return false
	}
	fw.etcdCtl.Client.SetLogger(&StdLogger{
		Name:        "ETCD",
		LogReplacer: strings.NewReplacer("[", "", "]", ""),
		LogWriter:   fw.wmLog,
	})

	// 注册自身
	var httpType = "https"
	if *Debug || *forceHTTP {
		httpType = "http"
	}
	if len(fw.rootPath) > 0 {
		fw.etcdCtl.Client.SetRoot(fw.rootPath)
	}
	a, b, err := net.SplitHostPort(fw.etcdCtl.regAddr)
	if err != nil {
		a = fw.etcdCtl.regAddr
	}
	if b == "" {
		b = strconv.Itoa(*webPort)
	}
	fw.etcdCtl.Client.Register(fw.serverName, a, b, httpType, "json")
	// 获取服务列表信息
	fw.etcdCtl.Client.Watcher()
	return true
}

// ETCDIsReady 返回ETCD可用状态
func (fw *WMFrameWorkV2) ETCDIsReady() bool {
	return fw.etcdCtl.enable
}

// Picker 选取服务地址
func (fw *WMFrameWorkV2) Picker(svrName string) (string, error) {
	if !fw.etcdCtl.enable {
		return "", fmt.Errorf("etcd client not ready")
	}
	addr, err := fw.etcdCtl.Client.Picker(svrName)
	if err != nil {
		return "", err
	}
	return addr, nil
}

// PickerDetail 选取服务地址,带http(s)前缀
func (fw *WMFrameWorkV2) PickerDetail(svrName string) (string, error) {
	if !fw.etcdCtl.enable {
		return "", fmt.Errorf("etcd client not ready")
	}
	addr, err := fw.etcdCtl.Client.PickerDetail(svrName)
	if err != nil {
		return "", err
	}
	return addr, nil
}

// ViewETCDConfig 查看ETCD配置,返回json字符串
func (fw *WMFrameWorkV2) ViewETCDConfig() string {
	return fw.etcdCtl.forshow
}
