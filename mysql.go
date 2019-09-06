package wlstmicro

import (
	"strconv"

	"db"

	"github.com/xyzj/gopsu"
)

var (
	// MysqlClient mysql连接池
	MysqlClient *db.MySQL
)

// NewMysqlClient mariadb client
func NewMysqlClient(mark string) {
	if AppConf == nil {
		WriteLog("SYS", "Configuration files should be loaded first", 40)
		return
	}
	mysqlConf.addr = AppConf.GetItemDefault("db_addr", "127.0.0.1:3306", "mysql服务地址,ip:port格式")
	mysqlConf.user = AppConf.GetItemDefault("db_user", "root", "mysql用户名")
	mysqlConf.pwd = gopsu.DecodeString(AppConf.GetItemDefault("db_pwd", "SsWAbSy8H1EOP3n5LdUQqls", "mysql密码"))
	mysqlConf.database = AppConf.GetItemDefault("db_name", "mydb1024", "mysql数据库名称")
	mysqlConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("db_enable", "true", "是否启用mysql"))

	if !mysqlConf.enable {
		return
	}
	var err error
	MysqlClient, err = db.GetNewDBPool(mysqlConf.user, mysqlConf.pwd, mysqlConf.addr, mysqlConf.database, 10, true, 30)
	if err != nil {
		WriteLog("SQL", "Failed connect to server "+mysqlConf.addr+"|"+err.Error(), 40)
		return
	}
	activeMysql = true
	WriteLog("SQL", "Success connect to server "+mysqlConf.addr, 90)

	MysqlClient.ConfigCache(gopsu.DefaultCacheDir, "um"+mark, 30)
}

// MysqlIsReady 返回mysql可用状态
func MysqlIsReady() bool {
	return mysqlConf != nil
}
