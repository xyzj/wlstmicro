package wmv2

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/db"
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
	// 使用mrg_myisam引擎的总表名称
	mrgTables []string
	// mrg_myisam引擎最大分表数量
	mrgMaxSubTables int
	// mrg_myisam分表大小（MB），默认1800
	mrgSubTableSize int64
	// mrg_myisam分表行数，默认4800000
	mrgSubTableRows int64
	// client
	client *db.SQLPool
}

func (conf *dbConfigure) show() string {
	conf.forshow, _ = sjson.Set("", "addr", conf.addr)
	conf.forshow, _ = sjson.Set(conf.forshow, "user", CWorker.Encrypt(conf.user))
	conf.forshow, _ = sjson.Set(conf.forshow, "pwd", CWorker.Encrypt(conf.pwd))
	conf.forshow, _ = sjson.Set(conf.forshow, "dbname", conf.database)
	conf.forshow, _ = sjson.Set(conf.forshow, "driver", conf.driver)
	conf.forshow, _ = sjson.Set(conf.forshow, "enable", conf.enable)
	return conf.forshow
}

// Newfw.dbCtl.client mariadb client
func (fw *WMFrameWorkV2) newDBClient() bool {
	fw.dbCtl.addr = fw.wmConf.GetItemDefault("db_addr", "127.0.0.1:3306", "sql服务地址,ip[:port[/instance]]格式")
	fw.dbCtl.user = fw.wmConf.GetItemDefault("db_user", "root", "sql用户名")
	fw.dbCtl.pwd = gopsu.DecodeString(fw.wmConf.GetItemDefault("db_pwd", "SsWAbSy8H1EOP3n5LdUQqls", "sql密码"))
	fw.dbCtl.database = fw.wmConf.GetItemDefault("db_name", "", "sql数据库名称")
	fw.dbCtl.driver = fw.wmConf.GetItemDefault("db_drive", "mysql", "sql数据库驱动，mysql 或 mssql")
	fw.dbCtl.enable, _ = strconv.ParseBool(fw.wmConf.GetItemDefault("db_enable", "true", "是否启用sql"))
	fw.wmConf.Save()
	fw.dbCtl.show()
	if !fw.dbCtl.enable {
		return false
	}
	fw.dbCtl.client = &db.SQLPool{
		User:         fw.dbCtl.user,
		Server:       fw.dbCtl.addr,
		Passwd:       fw.dbCtl.pwd,
		DataBase:     fw.dbCtl.database,
		EnableCache:  true,
		MaxOpenConns: 200,
		CacheDir:     gopsu.DefaultCacheDir,
		Timeout:      120,
		Logger: &StdLogger{
			Name:        "SQL",
			LogReplacer: strings.NewReplacer("[", "", "]", ""),
			LogWriter:   fw.wmLog,
		},
	}
	switch fw.dbCtl.driver {
	case "mssql":
		fw.dbCtl.client.DriverType = db.DriverMSSQL
	default:
		fw.dbCtl.client.DriverType = db.DriverMYSQL
	}
	err := fw.dbCtl.client.New()
	if err != nil {
		fw.dbCtl.enable = false
		fw.WriteError("SQL", "Failed connect to server "+fw.dbCtl.addr+"|"+err.Error())
		return false
	}

	return true
}

// MaintainMrgTables 维护mrg引擎表
func (fw *WMFrameWorkV2) MaintainMrgTables() {
	// 延迟一下，确保sql已连接
	time.Sleep(time.Minute)
	if !fw.dbCtl.enable {
		return
	}
MAINTAIN:
	func() {
		defer func() {
			if err := recover(); err != nil {
				fw.WriteError("SQL", err.(error).Error())
			}
		}()
		for {
			t := time.Now()
			if t.Minute() == 1 && t.Hour() == 2 {
				// 重新刷新配置
				fw.dbCtl.mrgTables = strings.Split(fw.wmConf.GetItemDefault("db_mrg_tables", "", "使用mrg_myisam引擎分表的总表名称，用`,`分割多个总表"), ",")
				fw.dbCtl.mrgMaxSubTables = gopsu.String2Int(fw.wmConf.GetItemDefault("db_mrg_maxsubtables", "10", "分表子表数量，最小为1"), 10)
				fw.dbCtl.mrgSubTableSize = gopsu.String2Int64(fw.wmConf.GetItemDefault("db_mrg_subtablesize", "1800", "子表最大磁盘空间容量（MB），当超过该值时，进行分表操作,推荐默认值1800"), 10)
				if fw.dbCtl.mrgSubTableSize < 1 {
					fw.dbCtl.mrgSubTableSize = 10
				}
				fw.dbCtl.mrgSubTableRows = gopsu.String2Int64(fw.wmConf.GetItemDefault("db_mrg_subtablerows", "4500000", "子表最大行数，当超过该值时，进行分表操作，推荐默认值4500000"), 10)

				for _, v := range fw.dbCtl.mrgTables {
					tableName := strings.TrimSpace(v)
					if tableName == "" {
						continue
					}
					_, _, size, rows, err := fw.dbCtl.client.ShowTableInfo(tableName)
					if err != nil {
						fw.WriteError("SQL", "SHOW table "+tableName+" "+err.Error())
						continue
					}
					if size >= fw.dbCtl.mrgSubTableSize || rows >= fw.dbCtl.mrgSubTableRows {
						err = fw.dbCtl.client.MergeTable(tableName, fw.dbCtl.mrgMaxSubTables)
						if err != nil {
							fw.WriteError("SQL", "MRG table "+tableName+" "+err.Error())
							continue
						}
					}
				}
				time.Sleep(time.Hour)
			}
			time.Sleep(time.Second * 30)
		}
	}()
	time.Sleep(time.Minute)
	goto MAINTAIN
}

// MysqlIsReady 返回mysql可用状态
func (fw *WMFrameWorkV2) MysqlIsReady() bool {
	return fw.dbCtl.enable
}

// ViewSQLConfig 查看sql配置,返回json字符串
func (fw *WMFrameWorkV2) ViewSQLConfig() string {
	return fw.dbCtl.forshow
}

// DBUpgrade 检查是否需要升级数据库
//  返回是否执行过升级，true-执行了升级，false-不需要升级
func (fw *WMFrameWorkV2) DBUpgrade(sql []byte) bool {
	if !fw.dbCtl.enable || sql == nil {
		return false
	}
	// 检查升级文件
	upsql := filepath.Join(gopsu.GetExecDir(), gopsu.GetExecName()) + ".dbupg"
	// 校验升级脚本
	b, _ := ioutil.ReadFile(upsql)
	if string(b) == gopsu.GetMD5((string(sql))) { // 升级脚本已执行过，不再重复升级
		return false
	}
	// 执行升级脚本
	var err error
	fw.WriteInfo("DBUP", "Try to update database")
	for _, v := range strings.Split(string(sql), ";") {
		s := strings.TrimSpace(v)
		if s == "" {
			continue
		}
		if _, _, err = fw.dbCtl.client.Exec(s + ";"); err != nil {
			if strings.Contains(err.Error(), "Duplicate") {
				continue
			}
			fw.WriteError("DBUP", s+" | "+err.Error())
		}
	}
	// 标记脚本，下次启动不再重复升级
	ioutil.WriteFile(upsql, []byte(gopsu.GetMD5(string(sql))), 0664)
	return true
}
