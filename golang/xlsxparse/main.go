package main

import (
	"flag"
	"path/filepath"
)

var xlsxRoot string   //xlsx配置路径
var curAbsRoot string //当前路径
var luaAbsRoot string //lua输出路径
var tsAbsRoot string  //ts输出路径

func main() {
	var err error
	curAbsRoot, err = filepath.Abs("./")
	if err != nil {
		panic(err)
	}

	flag.StringVar(&xlsxRoot, "i", "xlsx", "输入路径")
	pLua := flag.String("lua", "l-xlsx", "lua 输出路径")
	pTs := flag.String("ts", "t-xlsx", "ts 输出路径")
	flag.Parse()
	luaAbsRoot = curAbsRoot + "\\" + *pLua
	tsAbsRoot = curAbsRoot + "\\" + *pTs

	startConv()
}
