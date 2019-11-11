package wlstmicro

import (
	"strconv"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/db"
)

var (
	// MysqlClient mysql连接池
	MysqlClient *db.MySQL
)

// NewMysqlClient mariadb client
func NewMysqlClient(mark string) bool {
	if AppConf == nil {
		WriteError("SYS", "Configuration files should be loaded first")
		return false
	}
	dbConf.addr = AppConf.GetItemDefault("db_addr", "127.0.0.1:3306", "sql服务地址,ip[:port[/instance]]格式")
	dbConf.user = AppConf.GetItemDefault("db_user", "root", "sql用户名")
	dbConf.pwd = gopsu.DecodeString(AppConf.GetItemDefault("db_pwd", "SsWAbSy8H1EOP3n5LdUQqls", "sql密码"))
	dbConf.database = AppConf.GetItemDefault("db_name", "mydb1024", "sql数据库名称")
	dbConf.driver = AppConf.GetItemDefault("db_drive", "mysql", "sql数据库驱动，mysql 或 mssql")
	// dbConf.usetls, _ = strconv.ParseBool(AppConf.GetItemDefault("etcd_tls", "false", "是否使用证书连接sql服务"))
	dbConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("db_enable", "true", "是否启用sql"))

	if !dbConf.enable {
		return false
	}
	var err error
	switch dbConf.driver {
	case "mssql":
		db.SetDBDriver(db.DriverMSSQL)
	default:
		db.SetDBDriver(db.DriverMYSQL)
	}
	MysqlClient, err = db.GetNewDBPool(dbConf.user, dbConf.pwd, dbConf.addr, dbConf.database, 10, true, 30)
	if err != nil {
		WriteError("SQL", "Failed connect to server "+dbConf.addr+"|"+err.Error())
		return false
	}
	WriteSystem("SQL", "Success connect to server "+dbConf.addr)

	MysqlClient.ConfigCache(gopsu.DefaultCacheDir, "gc"+mark, 30)
	return true
}

// MysqlIsReady 返回mysql可用状态
func MysqlIsReady() bool {
	if MysqlClient != nil {
		return MysqlClient.IsReady
	}
	return false
}
