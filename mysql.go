package wlstmicro

import (
	"strconv"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/db"
)

var (
	mysqlClient *db.MySQL
)

// NewMysqlClient mariadb client
func NewMysqlClient(mark string) {
	if appConf == nil {
		writeLog("SYS", "Configuration files should be loaded first", 40)
		return
	}
	mysqlConf.addr = appConf.GetItemDefault("db_addr", "127.0.0.1:3306", "mysql服务地址,ip:port格式")
	mysqlConf.user = appConf.GetItemDefault("db_user", "root", "mysql用户名")
	mysqlConf.pwd = gopsu.DecodeString(appConf.GetItemDefault("db_pwd", "SsWAbSy8H1EOP3n5LdUQqls", "mysql密码"))
	mysqlConf.database = appConf.GetItemDefault("db_name", "mydb1024", "mysql数据库名称")
	if !standAloneMode {
		mysqlConf.enable, _ = strconv.ParseBool(appConf.GetItemDefault("db_enable", "true", "是否启用mysql"))
	}

	if !mysqlConf.enable {
		return
	}
	var err error
	mysqlClient, err = db.GetNewDBPool(mysqlConf.user, mysqlConf.pwd, mysqlConf.addr, mysqlConf.database, 10, true, 30)
	if err != nil {
		writeLog("SQL", "Failed connect to server "+mysqlConf.addr+"|"+err.Error(), 40)
		return
	}
	activeMysql = true
	writeLog("SQL", "Success connect to server "+mysqlConf.addr, 90)

	mysqlClient.ConfigCache(gopsu.DefaultCacheDir, "um"+mark, 30)
}

// MysqlIsReady 返回mysql可用状态
func MysqlIsReady() bool {
	return mysqlConf != nil
}
