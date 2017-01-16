package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type Result struct {
	Res string
	Num int
}

// 数据库
type DataBase struct {
	Name    string           // 数据库名称
	Conn    *sql.DB          // 数据库连接
	Tables  map[string]Table // 数据库中所有的表
	ClearOk bool             // 数据库是否清理成功
	C       chan int         // channel

	// 私有
	queue    []string          // goroutine处理队列
	schedule map[int]int       // goroutine处理进度
	index    int               // 接下来要处理的索引
	clearRes map[string]Result // 清理结果
	mergeRes map[string]Result // 合并结果
}

type FieldInfo map[string]string // 字段的信息
// 表结构信息
type Table struct {
	Name   string   //表名
	PriKey []string //主键
	Fields []FieldInfo

	hasPid bool // 是否有pid字段
}

func InitDB(dbName string) *DataBase {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", conf.User, conf.Password, conf.Address, dbName)
	db, err := sql.Open("mysql", dsn)
	checkError(err)
	err = db.Ping()
	checkError(err)

	gameDB := DataBase{Name: dbName, Conn: db, ClearOk: true}
	gameDB.Tables = make(map[string]Table, 100)
	gameDB.C = make(chan int, conf.Worker)

	gameDB.clearRes = make(map[string]Result, 100)
	gameDB.mergeRes = make(map[string]Result, 100)
	gameDB.queue = make([]string, 0, 100)
	gameDB.schedule = make(map[int]int, 100)

	// 查询库中所有table
	gameDB.DropTempTable()
	gameDB.ParseAllTables()
	gameDB.CreateTempTable()

	return &gameDB
}

// 创建一个临时表 temp_for_merge
func (this *DataBase) CreateTempTable() {
	_, err := this.Conn.Exec("CREATE TABLE `temp_for_merge` (" +
		"`pid` int(11) unsigned NOT NULL COMMENT 'pid'," +
		"PRIMARY KEY (`pid`)" +
		") ENGINE=MyISAM AUTO_INCREMENT=5 DEFAULT CHARSET=utf8 COMMENT='临时表'")
	checkError(err)
}

// 删除临时表
func (this *DataBase) DropTempTable() {
	// drop table temp_for_merge
	_, err := this.Conn.Exec("drop table if exists temp_for_merge ")
	checkError(err)
}

// 是否是不需要处理的表
func (this *DataBase) isExclude(tbName string) bool {
	for _, v := range conf.Exclude {
		if tbName == v {
			return true
		}
	}
	return false
}

func (this *DataBase) ParseAllTables() {
	rows, err := this.Conn.Query("show tables")
	checkError(err)

	var name string // 表名
	for rows.Next() {
		err = rows.Scan(&name)
		checkError(err)

		this.ParseTable(name)
	}
}

// 解析table表结构
func (this *DataBase) ParseTable(tbName string) {
	rows, err := this.Conn.Query("desc " + tbName)
	checkError(err)

	// 字段
	columns, cErr := rows.Columns()
	checkError(cErr)

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	table := Table{Name: tbName, hasPid: false}
	table.PriKey = make([]string, 0, 5)      //最多5个主键
	table.Fields = make([]FieldInfo, 0, 100) //最多100个字段
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		checkError(cErr)

		info := make(map[string]string)
		var value string
		for i, col := range values {
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			info[columns[i]] = value
		}

		if v, ok := info["Key"]; ok && v == "PRI" { //是否是主键
			table.PriKey = append(table.PriKey, info["Field"])
		}
		if info["Field"] == "pid" {
			table.hasPid = true
		}
		table.Fields = append(table.Fields, info)
	}

	err = rows.Err()
	checkError(err)

	this.Tables[tbName] = table
}

// 找到需要删除的死号玩家
func (this *DataBase) FindAndClear() {
	defer func() {
		if err := recover(); err != nil {
			this.ClearOk = false
			fmt.Printf("[%s]数据库清理错误，err=%s\n", this.Name, err)
		}
		wg.Done()
	}()

	// 超过30天没用上线的就做为死号
	zeroTm := GetTodayZeroTime()
	limitTm := zeroTm - 86400*30

	res, err := this.Conn.Exec("insert into temp_for_merge(pid) select pid from player where lastLoginTime<?", limitTm)
	checkError(err)

	num, _ := res.RowsAffected()
	if num > 0 { // 有无用数据
		for _, t := range this.Tables {
			if t.Name != "player" &&
				!this.isExclude(t.Name) &&
				t.hasPid {
				this.queue = append(this.queue, t.Name)
			}
		}

		// 开启goroutine
		num := len(this.queue)
		if num >= conf.Worker {
			num = conf.Worker
		}
		for i := 0; i < num; i++ {
			go this.ClearByIdx(this.index)
			this.index++
		}

	forLabel:
		for {
			select {
			case idx := <-this.C:
				this.schedule[idx] = 1
				if len(this.schedule) >= len(this.queue) {

					break forLabel
				} else {
					if this.index < len(this.queue) {
						go this.ClearByIdx(this.index)
						this.index++
					}
				}
			}
		}

		// 清理有依赖关系的表
		this.ClearSpecial()

		// 最后清player表
		resNum := this.clear("player")
		this.clearRes["player"] = Result{"OK", resNum}
	}
}

func (this *DataBase) clear(tbName string) int {
	sql := fmt.Sprintf("DELETE %s FROM %s,temp_for_merge WHERE %s.pid=temp_for_merge.pid", tbName, tbName, tbName)
	res, err := this.Conn.Exec(sql)
	checkError(err)

	var nums int64 = 0
	nums, err = res.RowsAffected()
	checkError(err)
	return int(nums)
}

// 开始清除每个库的无用数据
func (this *DataBase) ClearByIdx(tbIdx int) {
	// 每个goroutine捕获自己的panic
	defer func() {
		if err := recover(); err != nil {
			this.ClearOk = false
			tbName := this.queue[tbIdx]
			this.clearRes[tbName] = Result{fmt.Sprintf("%v", err), 0}
		}
		this.C <- tbIdx
	}()

	tbName := this.queue[tbIdx]
	num := this.clear(tbName)
	this.clearRes[tbName] = Result{"OK", num}
}

// 清理需要特殊处理的表
/* 清理公会成员思路
整个表按照office升序排列，再对排序结果按gid group by分组，
这样每一行数据就表示该公会目前职位最高的玩家。
最后对office不是1的更新为1。即找到该公会目前职位最高的玩家，并把他设为帮主。
*/
func (this *DataBase) ClearSpecial() {
	for _, t := range conf.Special {
		if _, ok := this.Tables[t.Name]; ok {
			num := this.ExecSqls(t)
			this.clearRes[t.Name] = Result{"OK", num}
		}
	}
}

func (this *DataBase) ExecSqls(t *SpecialTable) int {
	num := len(t.Sql)
	if num > 0 {
		if num == 1 {
			res, err := this.Conn.Exec(t.Sql[0])
			checkError(err)

			var nums int64 = 0
			nums, err = res.RowsAffected()
			checkError(err)
			return int(nums)
		} else if num > 1 {
			//开启事务
			tx, err := this.Conn.Begin()
			checkError(err)

			//出异常回滚
			defer tx.Rollback()

			//使用tx
			for _, sql := range t.Sql {
				tx.Exec(sql)
			}

			//提交事务
			err = tx.Commit()
			checkError(err)
		}
	}
	return 0
}

func (this *DataBase) reset() {
	// 清空工作队列
	this.queue = this.queue[0:0]
	for k, _ := range this.schedule {
		delete(this.schedule, k)
	}
	this.index = 0

	// 情空合并结果
	for k, _ := range this.mergeRes {
		delete(this.mergeRes, k)
	}
}

// 主数据库吃掉从数据库
func (this *DataBase) MergeDB(dbSlave *DataBase) {
	this.reset()

	for _, t := range this.Tables {
		if t.Name != "player" &&
			!this.isExclude(t.Name) {
			this.queue = append(this.queue, t.Name)
		}
	}

	// 首先合并player表
	fmt.Println("尝试合并player表")
	resNum := this.checkAndMerge(dbSlave, "player")
	this.mergeRes["player"] = Result{"OK", resNum}

	// 开启goroutine
	num := len(this.queue)
	if num >= conf.Worker {
		num = conf.Worker
	}
	for i := 0; i < num; i++ {
		go this.TryMerge(dbSlave, this.index)
		this.index++
	}

forLabel:
	for {
		select {
		case idx := <-this.C:
			this.schedule[idx] = 1
			if len(this.schedule) >= len(this.queue) {
				break forLabel
			} else {
				if this.index < len(this.queue) {
					go this.TryMerge(dbSlave, this.index)
					this.index++
				}
			}
		}
	}

	// 打印合并结构
	for tbName, res := range this.mergeRes {
		fmt.Printf("合并结果：表=%-20s,err=%s,num=%d\n", tbName, res.Res, res.Num)
	}
}

func (this *DataBase) checkAndMerge(dbSlave *DataBase, tbName string) int {
	tbMaster := this.Tables[tbName]
	tbSlave := dbSlave.Tables[tbName]
	master, slave := this.Name, dbSlave.Name
	//fmt.Printf("开始合并：[%s.%s] <-- [%s.%s]\n", master, tbName, slave, tbName)

	if len(tbMaster.Fields) != len(tbSlave.Fields) {
		panic("两个表的字段数量不一致")
	}

	// 每个字段的属性是否一致
	for i, f := range tbMaster.Fields {
		for k, info := range f {
			if v, ok := tbSlave.Fields[i][k]; ok {
				if v != info {
					panic("字段属性不一样")
				}
			} else {
				panic("从数据的table不存在字段：" + k)
			}
		}
	}

	// 检查主键数据是否有冲突
	// select COUNT(*) from domi.player as a,domi_temp.player as b where a.pid=b.pid;
	/*
		whereSlice := make([]string, 0, len(tbMaster.PriKey))
		for _, key := range tbMaster.PriKey {
			whereSlice = append(whereSlice, fmt.Sprintf(" a.%s=b.%s ", key, key))
		}
		where := strings.Join(whereSlice, "and")
		master, slave := this.Name, dbSlave.Name
		sql := fmt.Sprintf("select COUNT(*) from %s.%s as a,%s.%s as b where %s", master, tbName, slave, tbName, where)
		rows, err := this.Conn.Query(sql)
		checkError(err)
		fmt.Println(sql)

		var count int = 0
		for rows.Next() {
			err = rows.Scan(&count)
			checkError(err)
		}
		if count > 0 {
			errMsg := fmt.Sprintf("Error:主键元素有重复,冲突数量=%d", count)
			panic(errMsg)
		}
	*/

	// 开始合并,使用事务
	tx, err := this.Conn.Begin()
	defer tx.Rollback()
	checkError(err)

	sql := fmt.Sprintf("insert into %s.%s select * from %s.%s", master, tbName, slave, tbName)
	res, txErr := tx.Exec(sql)
	checkError(txErr)

	num, resErr := res.RowsAffected()
	checkError(resErr)
	if num > 0 {
		// 清空从数据库的表
		sql = fmt.Sprintf("delete from %s.%s", slave, tbName)
		_, err = tx.Exec(sql)
		checkError(err)
	}

	err = tx.Commit() // 提交事务
	checkError(err)

	return int(num)
}

// 表示吃掉dbSlave
func (this *DataBase) TryMerge(dbSlave *DataBase, tbIdx int) {
	defer func() {
		if err := recover(); err != nil {
			tbName := this.queue[tbIdx]
			errMsg := fmt.Sprintf("%v", err)
			this.mergeRes[tbName] = Result{errMsg, 0}
		}
		this.C <- tbIdx
	}()

	tbName := this.queue[tbIdx]
	num := this.checkAndMerge(dbSlave, tbName)
	this.mergeRes[tbName] = Result{"OK", num}
}
