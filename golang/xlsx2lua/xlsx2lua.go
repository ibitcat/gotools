/*
author：domicat
date： 2017-2-14
brief：xlsx转lua工具，支持id重复检测，支持json格式错误检测。
*/

package main

import (
	"bufio"
	//"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Luxurioust/excelize"
	"github.com/fatih/color"
)

const (
	E_NONE = iota
	E_WARN
	E_ERROR
)

func GetFullPath(path string) string {
	absolutePath, _ := filepath.Abs(path)
	return absolutePath
}

// 获取两个时间戳的毫秒差
func GetDurationMs(t time.Time) int {
	return int(time.Now().Sub(t).Nanoseconds() / 1e6)
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}

// 获取文件名
func getFileName(file string) string {
	return strings.TrimSuffix(file, ".xlsx")
}

// 获取lua输出路径（文件夹，例如：/item/）
func genLuaDir(xlsxpath string, file string) string {
	p := strings.TrimSuffix(xlsxpath, file)
	p = strings.TrimPrefix(p, xlsxRoot)
	luaDir := luaRoot + p
	return luaDir
}

// 删除服务器无用的lua文件
func removeLua(xlsxpath string, file string) {
	fileName := getFileName(file)
	luaDir := genLuaDir(xlsxpath, file)
	luaFile := fmt.Sprintf("%s%s.lua", luaDir, fileName)
	os.Remove(luaFile)
}

func loadLang(xlsxpath string) ([][]string, map[string]int, map[string]int) {
	p := strings.TrimPrefix(xlsxpath, xlsxRoot)
	sep := "/"
	if runtime.GOOS == "windows" {
		sep = "\\"
	}
	p = strings.Replace(p, sep, "$", -1)
	langFile := langRoot + sep + p

	xlFile, err := excelize.OpenFile(langFile)
	if err != nil {
		return nil, nil, nil
	}

	workRows := xlFile.GetRows("Sheet1")
	FieldRef := make(map[string]int)
	IdRef := make(map[string]int)
	for i, row := range workRows {
		if i == 0 { //第一行
			for j, text := range row {
				if strings.Contains(text, "_翻译") {
					fieldName := strings.TrimRight(text, "_翻译")
					if _, ok := FieldRef[fieldName]; ok {
						FieldRef[fieldName] = j
					}
				} else {
					FieldRef[text] = j
				}
			}
		} else {
			id := row[0]
			IdRef[id] = i
		}
	}

	return workRows, FieldRef, IdRef
}

func loadXlsx(xlsxpath string, file string) {
	errMsg := "OK"
	level := E_NONE
	defer func() {
		if err := recover(); err != nil {
			errMsg = fmt.Sprintf("%v", err)
		}
		resChan <- Result{xlsxpath, errMsg, level}
	}()

	xlFile, err := excelize.OpenFile(xlsxpath)
	if err != nil {
		level, errMsg = E_ERROR, err.Error()
		return
	}

	Key := ""
	Field := make(map[int]FieldInfo, 50) //最多50个字段

	// 第一个sheet为配置
	workRows := xlFile.GetRows("Sheet1")
	if len(workRows) < 4 {
		level, errMsg = E_ERROR, "至少要4行"
		return
	}

	fieldRow := workRows[1] //字段名
	typeRow := workRows[2]  //字段类型
	modeRow := workRows[3]  //生成方式
	if len(fieldRow) == 0 {
		level, errMsg = E_ERROR, "字段名为空"
		return
	}

	// 解析配置头
	var needTrans bool = false // 是否需要翻译
	var checkOnly bool = false // 纯客户端的配置，只需要检查id重复和json格式
	for i, fieldName := range fieldRow {
		fieldType := typeRow[i]
		modeType := modeRow[i]
		if i == 0 { //字段行的第一个字段为配置的key,需要检查下
			if len(fieldName) == 0 {
				level, errMsg = E_ERROR, "配置没有key"
				return
			}
			if modeType != "s" && modeType != "d" {
				level, errMsg = E_WARN, "不需要生成"
				checkOnly = true
				removeLua(xlsxpath, file)
			} else {
				if fieldType != "int" {
					level, errMsg = E_ERROR, "id字段类型必须为int"
					return
				}
			}

			Key = fieldName
		}

		if len(fieldName) > 0 {
			if strings.Contains(fieldName, " ") {
				level, errMsg = E_ERROR, fmt.Sprintf("字段名[%s]有空格", fieldName)
				return
			}
			if modeType != "c" && modeType != "s" && modeType != "d" {
				level, errMsg = E_ERROR, fmt.Sprintf("字段[%s]生成方式错误", fieldName)
				return
			}
			if len(fieldType) == 0 {
				level, errMsg = E_ERROR, "字段类型不存在"
				return
			}
		} else {
			if len(modeType) > 0 || len(fieldType) > 0 {
				level, errMsg = E_ERROR, fmt.Sprintf("第%d个字段名为空", i+1)
			}
			return
		}

		Field[i] = FieldInfo{fieldName, fieldType, modeType}
		if (modeType == "s" || modeType == "d") &&
			fieldType == "string" &&
			len(langRoot) > 0 {
			needTrans = true
		}
	}

	var fieldRef, idRef map[string]int
	var tranRows [][]string
	if needTrans {
		// 读取翻译文件
		tranRows, fieldRef, idRef = loadLang(xlsxpath)
	}

	// 配置
	sliceLen := len(workRows) * (len(Field) + 2)
	rowsSlice := make([]string, 0, sliceLen)
	if !checkOnly {
		// 文件头
		rowsSlice = append(rowsSlice, "--Don't Edit!!!") //第一行先占用

		// 配置table
		rowsSlice = append(rowsSlice, "return {")
		rowsSlice = append(rowsSlice, "{")
	}

	idMap := make(map[string]bool) //检查id是否重复
forLable:
	for i := 4; i < len(workRows); i++ {
		row := workRows[i]
		if len(row) > 0 {
			var id string
			for idx, text := range row {
				if idx == 0 { //key
					id = text
					if len(text) == 0 { //如果key字段的值为空，则停止解析配置
						break forLable
					} else {
						if _, ok := idMap[text]; ok {
							level, errMsg = E_ERROR, fmt.Sprintf("id重复,第%d行,id=%s", i+1, text)
							return
						} else {
							idMap[text] = true
						}
						if !checkOnly {
							str := fmt.Sprintf("    [%s] = {", text)
							rowsSlice = append(rowsSlice, str)
						}
					}
				}

				// fields
				if f, ok := Field[idx]; ok && len(text) > 0 {
					if f.Type == "table" {
						var temp interface{}
						err = json.Unmarshal([]byte(text), &temp)
						if err != nil {
							level, errMsg = E_ERROR, fmt.Sprintf("json格式错误,第%d行,字段%s", i+1, f.Name)
							return
						}
					}

					if !checkOnly && (f.Mode == "s" || f.Mode == "d") {
						var str string
						if f.Type == "int" || f.Type == "number" {
							str = fmt.Sprintf("        ['%s'] = %s,", f.Name, text)
						} else {
							if f.Type == "table" {
								// 压缩成一行
								text = strings.Replace(text, " ", "", -1)
								text = strings.Replace(text, "\n", "", -1)
							} else if f.Type == "string" { // ' 替换成 \'
								if needTrans {
									rId, rOk := idRef[id]
									cId, cOk := fieldRef[f.Name]
									if rOk && cOk &&
										len(tranRows) > rId &&
										len(tranRows[rId]) > cId {
										trCell := tranRows[rId][cId]
										if len(trCell) > 0 {
											text = trCell
										}
									}
								}
								text = strings.Replace(text, "'", `\'`, -1)
							}
							str = fmt.Sprintf("        ['%s'] = '%s',", f.Name, text)
						}
						rowsSlice = append(rowsSlice, str)
					}
				}
			}
			if !checkOnly {
				rowsSlice = append(rowsSlice, "    },")
			}
		}
	}
	if checkOnly {
		return
	}
	rowsSlice = append(rowsSlice, "},")

	// 字段table
	idxs := make([]int, 0, len(Field))
	for k, _ := range Field {
		idxs = append(idxs, k)
	}
	sort.Ints(idxs)
	rowsSlice = append(rowsSlice, "\n{")
	for _, v := range idxs {
		f := Field[v]
		if f.Mode == "s" || f.Mode == "d" {
			rowsSlice = append(rowsSlice, fmt.Sprintf("    ['%s'] = '%s',", f.Name, f.Type))
		}
	}
	rowsSlice = append(rowsSlice, "},\n")

	// key
	rowsSlice = append(rowsSlice, fmt.Sprintf("'%s'", Key))
	rowsSlice = append(rowsSlice, "}")

	// 生成lua文件
	outPutToLua(xlsxpath, file, rowsSlice)
}

// 目录不存在则新建目录
func outPutToLua(xlsxpath string, file string, rowsSlice []string) {
	fileName := getFileName(file)
	luaDir := genLuaDir(xlsxpath, file)

	_, err := os.Stat(luaDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(luaDir, os.ModePerm)
		if err != nil {
			panic("mkdir failed!")
			return
		}
	}

	//md5检测
	/*
		luaStr := strings.Join(rowsSlice, "\n")
		md5h := md5.New()
		md5h.Write([]byte(luaStr))
		md5str := fmt.Sprintf("%x", md5h.Sum([]byte("")))
		if v, ok := luaMap[p+fileName]; ok && md5str == v {
			panic("内容没有变化")
		} else {
			rowsSlice[0] = fmt.Sprintf("--md5:%s", md5str)
		}
	*/

	luaFile := fmt.Sprintf("%s%s.lua", luaDir, fileName)
	outFile, operr := os.OpenFile(luaFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	checkErr(operr)
	defer outFile.Close()

	outFile.WriteString(strings.Join(rowsSlice, "\n"))
	outFile.Sync()
}

func walkXlsx(path string) {
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		ok, mErr := filepath.Match("[^~$]*.xlsx", f.Name())
		if ok {
			if f == nil {
				return err
			}
			if f.IsDir() {
				return nil
			}

			xlsxMap[path] = f.Name()
			return nil
		}
		return mErr
	})
	checkErr(err)
}

func walkLua(path string) {
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		ok, mErr := filepath.Match("*.lua", f.Name())
		if ok {
			if f == nil {
				return err
			}
			if f.IsDir() {
				return nil
			}

			// 按行读取文件
			file, ferr := os.Open(path)
			checkErr(ferr)
			defer file.Close()

			scanner := bufio.NewScanner(file)
			if scanner.Scan() {
				firstLine := scanner.Text()
				if strings.Contains(firstLine, "--md5:") {
					p := strings.TrimPrefix(path, luaRoot)
					p = strings.TrimSuffix(p, ".lua")
					luaMap[p] = firstLine[6:]
				}
			}

			return nil
		}
		return mErr
	})
	checkErr(err)
}

type Result struct {
	Name   string
	ErrMsg string
	Level  int //0=普通 1=警告 2=错误
}

type FieldInfo struct {
	Name string
	Type string
	Mode string
}

var luaRoot string
var xlsxRoot string
var langRoot string
var resChan chan Result
var xlsxMap map[string]string
var resMap map[string]Result
var luaMap map[string]string

func main() {
	flag.StringVar(&xlsxRoot, "i", "", "输入路径")
	flag.StringVar(&luaRoot, "o", "", "输出路径")
	flag.StringVar(&langRoot, "l", "", "翻译文件路径")
	flag.Parse()
	if len(luaRoot) == 0 || len(xlsxRoot) == 0 {
		color.Red("输入路径或输出路径为空")
		return
	}

	luaMap = make(map[string]string)
	xlsxMap = make(map[string]string)
	resMap = make(map[string]Result)

	startTime := time.Now()
	//walkLua(luaRoot)
	walkXlsx(xlsxRoot)
	resChan = make(chan Result, len(xlsxMap))
	for path, f := range xlsxMap {
		go loadXlsx(path, f)
	}

	fmt.Printf("%-60s %-s\n", "path", "err")
	fmt.Println(strings.Repeat("-", 65))
	allOk := true
	for i := 0; i < len(xlsxMap); i++ {
		res := <-resChan
		if res.Level == 1 { //警告
			color.Set(color.FgYellow)
			fmt.Printf("%-60s %-s\n", res.Name, res.ErrMsg)
			color.Unset()
		} else if res.Level == 2 { //错误
			if allOk {
				allOk = false
			}
			color.Set(color.FgRed)
			fmt.Printf("%-60s %-s\n", res.Name, res.ErrMsg)
			color.Unset()
		} else {
			fmt.Printf("%-60s %-s\n", res.Name, res.ErrMsg)
		}
	}

	if allOk {
		fmt.Printf("生成完毕，耗时=%d 毫秒 ！\n", GetDurationMs(startTime))
	} else {
		color.Set(color.FgRed)
		fmt.Printf("生成有错误，耗时=%d 毫秒 ！\n", GetDurationMs(startTime))
		color.Unset()
	}
}
