package main

import (
	"fmt"
	"sync"
	"time"
)

var tempTb = "temp_for_merge"
var wg sync.WaitGroup
var GameDbMap map[string]*DataBase

// 获取当天凌晨时间戳
func GetTodayZeroTime() int64 {
	t := time.Unix(1484134400, 0)
	year, month, day := t.Date()
	t = time.Date(year, month, day, 0, 0, 0, 0, time.Local)

	return t.Unix()
}

func checkError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

// 合并数据
func Merge(master, slave string) {
	//检查两个库的表数量是否一致
	//检查主数据库的表是否在从数据库也存在
	//排除无需合并的表
	//检查两个表结构是否一致
	//检查是否有主键冲突
	//合并数据
	var dbMaster, dbSlave *DataBase
	var ok bool
	dbMaster, ok = GameDbMap[master]
	if !ok {
		panic("Error:主数据库不存在")
	}

	dbSlave, ok = GameDbMap[slave]
	if !ok {
		panic("Error:从数据库不存在")
	}

	if len(dbMaster.Tables) != len(dbSlave.Tables) {
		panic("Error:主从数据库的表数量不一致")
	}
	for _, t := range dbMaster.Tables {
		if _, ok := dbSlave.Tables[t.Name]; !ok {
			str := fmt.Sprintf("Error:从数据库[%s]的表[%s]不存在", dbSlave.Name, t.Name)
			panic(str)
		}
	}

	dbMaster.MergeDB(dbSlave)
}

func CloseDB() {
	for k, v := range GameDbMap {
		v.DropTempTable()
		v.Conn.Close()
		delete(GameDbMap, k)
	}
}

func main() {
	if !LoadJson() {
		return
	}

	GameDbMap = make(map[string]*DataBase)
	defer func() {
		CloseDB()
		if err := recover(); err != nil {
			fmt.Println("合服错误：", err)
		} else {
			fmt.Println("OK!合服结束")
		}
	}()

	// 初始化数据库连接
	GameDbMap[conf.MasterDb] = InitDB(conf.MasterDb)
	for _, name := range conf.SlaveDb {
		if _, ok := GameDbMap[name]; ok {
			panic("Error:数据库有重复")
		}
		GameDbMap[name] = InitDB(name)
	}

	// 查找每个库的无用数据并删除
	fmt.Println("---------------- clean db start ----------------")
	wg.Add(len(GameDbMap))
	for _, db := range GameDbMap {
		go db.FindAndClear()
	}
	wg.Wait()
	for _, db := range GameDbMap {
		if !db.ClearOk {
			fmt.Printf("Error：[%s]数据库未清理完成\n", db.Name)
			return
		}
	}
	fmt.Println("---------------- clean db end ----------------")
	fmt.Println()

	// 合并数据
	fmt.Println("---------------- merge db start ----------------")
	for _, name := range conf.SlaveDb {
		fmt.Printf("merge [%s]-[%s]: \n", conf.MasterDb, name)
		Merge(conf.MasterDb, name)
		fmt.Printf("merge [%s]-[%s] ok!!!\n\n", conf.MasterDb, name)
	}
	fmt.Println("---------------- merge db end ----------------")
}
