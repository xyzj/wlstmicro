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
func NewMysqlClient(mark string) {
	if AppConf == nil {
		WriteLog("SYS", "Configuration files should be loaded first", 40)
		return
	}
	dbConf.addr = AppConf.GetItemDefault("db_addr", "127.0.0.1:3306", "sql服务地址,ip[:port[/instance]]格式")
	dbConf.user = AppConf.GetItemDefault("db_user", "root", "sql用户名")
	dbConf.pwd = gopsu.DecodeString(AppConf.GetItemDefault("db_pwd", "SsWAbSy8H1EOP3n5LdUQqls", "sql密码"))
	dbConf.database = AppConf.GetItemDefault("db_name", "mydb1024", "sql数据库名称")
	dbConf.driver = AppConf.GetItemDefault("db_drive", "mysql", "sql数据库驱动，mysql 或 mssql")
	dbConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("db_enable", "true", "是否启用sql"))

	if !dbConf.enable {
		return
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
		WriteLog("SQL", "Failed connect to server "+dbConf.addr+"|"+err.Error(), 40)
		return
	}
	activeMysql = true
	WriteLog("SQL", "Success connect to server "+dbConf.addr, 90)

	MysqlClient.ConfigCache(gopsu.DefaultCacheDir, "um"+mark, 30)
}

// MysqlIsReady 返回mysql可用状态
func MysqlIsReady() bool {
	return dbConf != nil
}
