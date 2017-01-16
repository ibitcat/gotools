package main

import (
	"encoding/json"
	"io/ioutil"

	"fmt"
)

type Config struct {
	Worker   int             //工作线程数量
	MasterDb string          `json:"master_db"`
	SlaveDb  []string        `json:"slave_db"`
	Exclude  []string        //不需要合并的表
	Special  []*SpecialTable //需要特殊处理的表
}

type SpecialTable struct {
	Name string
	Sql  []string
}

var conf Config

func LoadJson() bool {
	data, err := ioutil.ReadFile("./config.json")
	checkError(err)

	err = json.Unmarshal(data, &conf)
	checkError(err)

	if len(conf.MasterDb) == 0 || len(conf.MasterDb) == 0 {
		fmt.Println("请输入主从数据库名")
		return false
	}
	for _, v := range conf.SlaveDb {
		if v == conf.MasterDb {
			fmt.Println("主从数据库不能为同一个数据库")
			return false
		}
	}
	return true
}
