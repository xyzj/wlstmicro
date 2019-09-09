package db

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	proto "github.com/golang/protobuf/proto"
	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
)

var (
	chanCloseCacheCheck = make(chan string, 2)
	startCacheCheck     = false
)

// MySQL 数据驱动封装
type MySQL struct {
	// ConnPool 数据库连接池
	ConnPool *sql.DB
	// IsReady 连接池是否就绪
	IsReady bool
	// 缓存路径
	cacheDir string
	// 缓存时间（分钟）0~60,0-表示不进行缓存操作
	cacheLife time.Duration
	// 缓存文件前缀
	cacheHead string
}

// GetNewDBPool 初始化连接池
//
// args:
//  username: 用户名
//  passwd: 密码
//  host: 数据库服务地址ip:port
//  dbname: 数据库表名，可为""
//  maxOpenConns: 连接池最大连接数量，范围0-200，idle数量为该值的一半，最低2个
//  multiStatements: 允许执行多条语句，true or false
//  readTimeout：I/O操作超时时间，单位秒，0-无超时
// return:
//  error
func GetNewDBPool(username, passwd, host, dbname string, maxOpenConns int, multiStatements bool, readTimeout uint32) (*MySQL, error) {
	var err error
	c := &MySQL{}
	if c.ConnPool, err = getMySQL(username, passwd, host, dbname, maxOpenConns, multiStatements, readTimeout); err == nil {
		c.IsReady = true
	} else {
		return nil, fmt.Errorf(fmt.Sprintf("DB open error: %+v", err))
	}
	return c, nil
}

// InitDBPool 初始化连接池（not recommend）
//
// args:
//  username: 用户名
//  passwd: 密码
//  host: 数据库服务地址ip:port
//  dbname: 数据库表名，可为""
//  maxOpenConns: 连接池最大连接数量，范围0-200，idle数量为该值的一半，最低2个
//  multiStatements: 允许执行多条语句，true or false
//  readTimeout：I/O操作超时时间，单位秒，0-无超时
// return:
//  error
func (c *MySQL) InitDBPool(username, passwd, host, dbname string, maxOpenConns int, multiStatements bool, readTimeout uint32) error {
	var err error
	if c.ConnPool, err = getMySQL(username, passwd, host, dbname, maxOpenConns, multiStatements, readTimeout); err == nil {
		c.IsReady = true
	} else {
		return errors.New(fmt.Sprintf("DB open error: %+v", err))
	}
	return nil
}

// checkSQL 检查sql语句是否存在注入攻击风险
//
// args：
//  s： sql语句
// return:
//  error
func (c *MySQL) checkSQL(s string) error {
	if gopsu.CheckSQLInject(s) {
		return nil
	} else {
		return fmt.Errorf("SQL statement has risk of injection.")
	}
}

// ConfigCache 配置缓存
//
// args:
//  s: 缓存文件夹路径
//  t: 缓存有效时间（0-60分钟）
// 当缓存有效时间 == 0时，表示不启用缓存功能，>0 时将启动一个后台线程，依据缓存有效时间定时清理过期缓存
func (c *MySQL) ConfigCache(cachedir, cachehead string, t int64) {
	if startCacheCheck {
		startCacheCheck = false
		chanCloseCacheCheck <- ""
	}
	c.cacheDir = cachedir
	c.cacheHead = cachehead
	if t > 60 {
		t = 60
	}
	if t < 0 {
		t = 0
	}
	c.cacheLife = time.Minute * time.Duration(t)
	if t > 0 {
		if startCacheCheck {
			return
		}
		startCacheCheck = true
		// 清理旧缓存
		go func() {
			defer func() {
				recover()
			}()
			for {
				select {
				case <-chanCloseCacheCheck:
					return
				case <-time.After(time.Minute * 5):
					c.checkCache()
				}
			}
		}()
	}
}

// 维护缓存文件数量
func (c *MySQL) checkCache() {
	files, err := ioutil.ReadDir(c.cacheDir)
	if err != nil {
		return
	}
	t := time.Now()
	for _, file := range files {
		ss := strings.Split(file.Name(), "#")
		if ss[0] != c.cacheHead {
			continue
		}
		if t.Sub(file.ModTime()).Minutes() > c.cacheLife.Minutes() {
			// if t-(gopsu.String2Int64(ss[1], 10)/60000000000) >= gopsu.String2Int64(ss[3], 10) {
			os.Remove(filepath.Join(c.cacheDir, file.Name()))
		}
	}
}

// 保留缓存
// func (c *MySQL) setCache(cacheTag string, cacheData *[]byte) {
// 	ioutil.WriteFile(filepath.Join(c.cacheDir, cacheTag), gopsu.DoZlibCompress([]byte(*cacheData)), 0444)
// }

// QueryCacheJSON 查询缓存结果
//
// args:
//  cacheTag: 缓存标签
//  startIdx: 起始行数
//  rowCount: 查询的行数
// return:
//  json字符串
func (c *MySQL) QueryCacheJSON(cacheTag string, startRow, rowsCount int) string {
	return string(gopsu.PB2Json(c.QueryCachePB2(cacheTag, startRow, rowsCount)))
}

// QueryCachePB2 查询缓存结果
//
// args:
//  cacheTag: 缓存标签
//  startIdx: 起始行数
//  rowCount: 查询的行数
// return:
//  &QueryData{}
func (c *MySQL) QueryCachePB2(cacheTag string, startRow, rowsCount int) *QueryData {
	if startRow < 1 {
		startRow = 1
	}
	if rowsCount < 1 {
		rowsCount = 1
	}
	query := &QueryData{CacheTag: cacheTag}
	if src, err := ioutil.ReadFile(filepath.Join(c.cacheDir, cacheTag)); err == nil {
		msg := &QueryData{}
		if ex := proto.Unmarshal(src, msg); ex == nil {
			query.Total = msg.Total
			startRow = startRow - 1
			endRow := startRow + rowsCount
			for k, v := range msg.Rows {
				if k >= startRow && k < endRow {
					query.Rows = append(query.Rows, v)
				}
				if k >= endRow {
					break
				}
			}
		}
	}
	return query
}

// QueryOne 执行查询语句，返回首行结果的json字符串，`{row：[...]}`，该方法不缓存结果
//
// args:
//  s: sql占位符语句
//  colNum: 列数量
//  params: 查询参数,语句中的参数用`?`占位
// return:
//  结果集json字符串，error
func (c *MySQL) QueryOne(s string, colNum int, params ...interface{}) (string, error) {
	var js string
	defer func() (string, error) {
		if err := recover(); err != nil {
			return "", err.(error)
		}
		return js, nil
	}()
	stmt, err := c.ConnPool.Prepare(s)
	if err != nil {
		return js, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(params...)
	if err != nil {
		return js, err
	}

	values := make([]interface{}, colNum)
	scanArgs := make([]interface{}, colNum)

	for i := range values {
		scanArgs[i] = &values[i]
	}

	err = row.Scan(scanArgs...)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", nil
		} else {
			return js, err
		}
	}
	for i := range scanArgs {
		v := values[i]
		b, ok := v.([]byte)
		if ok {
			js, _ = sjson.Set(js, "row.-1", string(b))
		} else {
			js, _ = sjson.Set(js, "row.-1", v)
		}
	}
	return js, nil
}

// QueryJSON 执行查询语句，返回结果集的json字符串
//
// args:
//  s: sql占位符语句
//  rowsCount: 返回数据行数，从第一行开始，0-返回全部
//  params: 查询参数,语句中的参数用`?`占位
// return:
//  结果集json字符串，error
func (c *MySQL) QueryJSON(s string, rowsCount int, params ...interface{}) (string, error) {
	x, ex := c.QueryPB2(s, rowsCount, params...)
	if ex != nil {
		return "", ex
	}
	return string(gopsu.PB2Json(x)), nil
}

// QueryPB2 执行查询语句，返回结果集的pb2序列化字节数组
//
// args:
//  s: sql占位符语句
//  rowsCount: 返回数据行数，从第一行开始，0-返回全部
//  params: 查询参数,语句中的参数用`?`占位
// return:
//  结果集的pb2序列化字节数组，error
func (c *MySQL) QueryPB2(s string, rowsCount int, params ...interface{}) (query *QueryData, err error) {
	defer func() (*QueryData, error) {
		if err := recover(); err != nil {
			return nil, err.(error)
		}
		return query, nil
	}()

	query = &QueryData{}
	queryCache := &QueryData{}

	stmt, err := c.ConnPool.Prepare(s)
	if err != nil {
		return query, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(params...)
	if err != nil {
		return query, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return query, err
	}
	count := len(columns)
	values := make([]interface{}, count)
	scanArgs := make([]interface{}, count)

	for i := range values {
		scanArgs[i] = &values[i]
	}

	var rowIdx = 0
	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			return query, err
		}
		row := &QueryData_Row{}
		for i := range columns {
			v := values[i]
			var cell interface{}
			b, ok := v.([]byte)
			if ok {
				cell = string(b)
			} else {
				cell = v
			}
			if cell == nil {
				cell = ""
			}
			row.Cells = append(row.Cells, fmt.Sprintf("%v", cell))
		}
		queryCache.Rows = append(queryCache.Rows, row)
		rowIdx++
	}
	if err := rows.Err(); err != nil {
		return query, err
	}
	query.Total = int32(rowIdx)
	queryCache.Total = int32(rowIdx)

	if rowsCount <= 0 || rowsCount > rowIdx {
		query.Rows = queryCache.Rows
	} else {
		query.Rows = append(query.Rows, queryCache.Rows[:rowsCount]...)
	}
	// 开始缓存
	if c.cacheLife > 0 && rowsCount > 0 && rowsCount < rowIdx {
		cacheTag := fmt.Sprintf("%s#%d#%d", c.cacheHead, time.Now().UnixNano(), rowIdx)
		query.CacheTag = cacheTag
		go func() {
			defer func() {
				recover()
			}()
			if b, ex := proto.Marshal(queryCache); ex == nil {
				ioutil.WriteFile(filepath.Join(c.cacheDir, cacheTag), b, 0444)
			}
		}()
	}
	return query, nil
}

// Exec 执行语句（insert，delete，update）,返回（影响行数,insertId,error）,使用官方的语句参数分离写法
//
// args:
//  s: sql占位符语句
//  param: 参数,语句中的参数用`?`占位
// return:
//   影响行数，insert的id，error
func (c *MySQL) Exec(s string, param ...interface{}) (int64, int64, error) {
	var lastID, rowCnt int64
	defer func() (int64, int64, error) {
		if err := recover(); err != nil {
			return 0, 0, err.(error)
		}
		return lastID, rowCnt, nil
	}()
	var stmt *sql.Stmt
	var err error
	if stmt, err = c.ConnPool.Prepare(s); err != nil {
		return 0, 0, err
	}
	defer stmt.Close()
	res, _ := stmt.Exec(param...)
	lastID, _ = res.LastInsertId()
	rowCnt, _ = res.RowsAffected()
	return rowCnt, lastID, nil
}

// ExecPrepare 批量执行语句（insert，delete，update），使用官方的语句参数分离写法，只能批量执行相同的语句
//
// args:
//  s: sql占位符语句
//  paramNum: 占位符数量,为0时自动计算sql语句中`?`的数量
//  params: 语句参数 `d := make([]interface{}, 0);d=append(d,xxx)`
// return:
//  error
func (c *MySQL) ExecPrepare(s string, paramNum int, params ...interface{}) error {
	defer func() error {
		if err := recover(); err != nil {
			return err.(error)
		}
		return nil
	}()
	if paramNum == 0 {
		paramNum = strings.Count(s, "?")
	}

	// 开启事务
	tx, err := c.ConnPool.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var stmt *sql.Stmt
	stmt, err = tx.Prepare(s)
	if err != nil {
		return err
	}
	defer stmt.Close()
	i := 0

	param := make([]interface{}, 0)
	for _, v := range params {
		i++
		param = append(param, v)
		if i == paramNum {
			_, err = stmt.Exec(param...)
			if err != nil {
				return err
			}
			param = make([]interface{}, 0)
			i = 0
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

// ExecBatch (maybe unsafe)批量执行语句（insert，delete，update）
//
// args：
//  s： sql语句组
// return:
//  error
func (c *MySQL) ExecBatch(s []string) error {
	defer func() error {
		if err := recover(); err != nil {
			return err.(error)
		}
		return nil
	}()
	// 检查语句，有任意语句存在风险，全部语句均不执行
	for _, v := range s {
		if err := c.checkSQL(v); err != nil {
			return err
		}
	}
	// 开启事务
	tx, err := c.ConnPool.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, v := range s {
		_, err = c.ConnPool.Exec(v)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

// GetDbConn 获取数据库连接实例，utf8字符集，连接超时10s
//  username: 数据库连接用户名
//  password： 数据库连接密码
//  host：主机名/主机ip
//  dbname：数据库名称，为空时表示不指定数据库
//  maxOpenConns：连接池中最大连接数，有效范围1-200，超范围时强制为20
//  multiStatements：允许执行多条语句，true or false
//  readTimeout：I/O操作超时时间，单位秒，0-无超时
func getMySQL(username, password, host, dbname string, maxOpenConns int, multiStatements bool, readTimeout uint32) (*sql.DB, error) {
	ms := "false"
	if multiStatements {
		ms = "true"
	}
	connString := fmt.Sprintf("%s:%s@tcp(%s)/%s"+
		"?multiStatements=%s"+
		"&readTimeout=%ds"+
		"&parseTime=true"+
		"&timeout=10s"+
		"&charset=utf8"+
		"&columnsWithAlias=true",
		username, password, host, dbname, ms, readTimeout)
	db, err := sql.Open("mysql", strings.Replace(connString, "\n", "", -1))

	if err != nil {
		return nil, err
	}

	if maxOpenConns <= 0 || maxOpenConns > 200 {
		maxOpenConns = 20
	}
	if maxOpenConns < 2 {
		db.SetMaxIdleConns(maxOpenConns)
	} else {
		db.SetMaxIdleConns(maxOpenConns / 2)
	}
	db.SetMaxOpenConns(maxOpenConns)

	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
