package wlstmicro

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/db"
)

var (
	dbConf = &dbConfigure{}
	// MysqlClient mysql连接池
	MysqlClient *db.SQLPool
)

// 数据库配置
type dbConfigure struct {
	forshow string
	// 数据库地址
	addr string
	// 登录用户名
	user string
	// 登录密码
	pwd string
	// 数据库名称
	database string
	// 数据库驱动模式，mssql/mysql
	driver string
	// 是否启用数据库
	enable bool
	// 是否启用tls
	usetls bool
	// 使用mrg_myisam引擎的总表名称
	mrgTables []string
	// mrg_myisam引擎最大分表数量
	mrgMaxSubTables int
	// mrg_myisam分表大小（MB），默认1800
	mrgSubTableSize int64
	// mrg_myisam分表行数，默认4800000
	mrgSubTableRows int64
}

func (conf *dbConfigure) show() string {
	conf.forshow, _ = sjson.Set("", "addr", dbConf.addr)
	conf.forshow, _ = sjson.Set(conf.forshow, "user", CWorker.Encrypt(dbConf.user))
	conf.forshow, _ = sjson.Set(conf.forshow, "pwd", CWorker.Encrypt(dbConf.pwd))
	conf.forshow, _ = sjson.Set(conf.forshow, "dbname", dbConf.database)
	conf.forshow, _ = sjson.Set(conf.forshow, "driver", dbConf.driver)
	conf.forshow, _ = sjson.Set(conf.forshow, "enable", dbConf.enable)
	return conf.forshow
}

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
	dbConf.mrgTables = strings.Split(AppConf.GetItemDefault("db_mrg_tables", "", "使用mrg_myisam引擎分表的总表名称，用`,`分割多个总表"), ",")
	dbConf.mrgMaxSubTables = gopsu.String2Int(AppConf.GetItemDefault("db_mrg_maxsubtables", "10", "分表子表数量，最小为1"), 10)
	dbConf.mrgSubTableSize = gopsu.String2Int64(AppConf.GetItemDefault("db_mrg_subtablesize", "1800", "子表最大磁盘空间容量（MB），当超过该值时，进行分表操作,推荐默认值1800"), 10)
	if dbConf.mrgSubTableSize < 1 {
		dbConf.mrgSubTableSize = 10
	}
	dbConf.mrgSubTableRows = gopsu.String2Int64(AppConf.GetItemDefault("db_mrg_subtablerows", "4500000", "子表最大行数，当超过该值时，进行分表操作，推荐默认值4500000"), 10)
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

// MaintainMrgTables 维护mrg引擎表
func MaintainMrgTables() {
	var mrgLocker sync.WaitGroup
MAINTAIN:
	go func() {
		defer func() {
			if err := recover(); err != nil {
				WriteError("SQL", err.(error).Error())
			}
			mrgLocker.Done()
		}()
		mrgLocker.Add(1)
		for {
			t := time.Now()
			if t.Minute() == 1 && t.Hour() == 2 {
				for _, v := range dbConf.mrgTables {
					tableName := strings.TrimSpace(v)
					if tableName == "" {
						continue
					}
					_, _, size, rows, err := MysqlClient.ShowTableInfo(tableName)
					if err != nil {
						WriteError("SQL", "SHOW table "+tableName+" "+err.Error())
						continue
					}
					if size >= dbConf.mrgSubTableSize || rows >= dbConf.mrgSubTableRows {
						err = MysqlClient.MergeTable(tableName, dbConf.mrgMaxSubTables)
						if err != nil {
							WriteError("SQL", "MRG table "+tableName+" "+err.Error())
							continue
						}
					}
				}
				time.Sleep(time.Hour)
			}
			time.Sleep(time.Second * 30)
		}
	}()
	for {
		time.Sleep(time.Minute)
		mrgLocker.Wait()
		goto MAINTAIN
	}
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
