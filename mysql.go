package wlstmicro

import (
	"strconv"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/db"
)

var (
	// MysqlClient mysql连接池
	MysqlClient *db.SQLPool
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
	dbConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("db_enable", "true", "是否启用sql"))
	AppConf.Save()
	dbConf.show()
	if !dbConf.enable {
		return false
	}
	MysqlClient = &db.SQLPool{
		User:        dbConf.user,
		Server:      dbConf.addr,
		Passwd:      dbConf.pwd,
		DataBase:    dbConf.database,
		EnableCache: true,
		CacheDir:    gopsu.DefaultCacheDir,
		CacheHead:   "gc" + mark,
		Timeout:     120,
		Logger: &StdLogger{
			Name: "SQL",
		},
	}
	switch dbConf.driver {
	case "mssql":
		MysqlClient.DriverType = db.DriverMSSQL
	default:
		MysqlClient.DriverType = db.DriverMYSQL
	}
	err := MysqlClient.New()
	if err != nil {
		WriteError("SQL", "Failed connect to server "+dbConf.addr+"|"+err.Error())
		return false
	}

	return true
}

// MysqlIsReady 返回mysql可用状态
func MysqlIsReady() bool {
	if MysqlClient != nil {
		return MysqlClient.IsReady()
	}
	return false
}

// ViewSSQLQLConfig 查看sql配置,返回json字符串
func ViewSSQLQLConfig() string {
	return dbConf.forshow
}
